package transport

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/morganhein/go-telnet"
	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

type juniper struct {
	ssh struct {
		Config     *ssh.ClientConfig
		connection *ssh.Client
		session    *ssh.Session
	}
	telnet struct {
		conn net.Conn
	}
	connOptions  schema.ConnectOptions
	ready        bool //set to false when running a command
	stdout       io.Reader
	stdin        io.WriteCloser
	stderr       io.Reader
	shutdown     chan bool //shutdown channel for the publisher
	continuation []*regexp.Regexp
	prompt       *regexp.Regexp
	events       chan schema.MessageEvent
	publisher    *pubsub.Publisher
	timeout      time.Duration  // The default timeout for this device
	attachWg     sync.WaitGroup // The waitgroup for the publisher attachment
}

func (j *juniper) Initialize() error {
	j.events = make(chan schema.MessageEvent, 20)
	j.publisher = pubsub.New(j, j.events)
	j.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`^.*?--More-- $`} {
		if re, err := regexp.Compile(next); err == nil {
			j.continuation = append(j.continuation, re)
		}
	}
	j.ready = false
	j.timeout = time.Duration(8) * time.Second
	return nil
}

func (j *juniper) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		return j.connectSsh(options)
	}
	if method == Telnet {
		options.Method = Telnet
		return j.connectTelnet(options)
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (j *juniper) SupportedMethods() []schema.ConnectionMethod {
	return []schema.ConnectionMethod{SSH, Telnet}
}

func (j *juniper) connectSsh(options schema.ConnectOptions) error {
	j.ssh.Config = CreateSSHConfig(options)
	//j.ssh.Config.Ciphers = []string{
	//	"aes128-cbc",
	//	"aes256-cbc",
	//	"aes128-ctr",
	//	"aes192-ctr",
	//	"aes256-ctr",
	//	"aes128-gcm@openssh.com",
	//	"arcfour256",
	//	"arcfour128",
	//}
	j.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	fmt.Println(j.ssh.Config.Ciphers)
	conn, err := ssh.Dial("tcp", host, j.ssh.Config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	j.ssh.connection = conn
	j.ssh.session, err = j.ssh.connection.NewSession()
	if err != nil {
		fmt.Errorf("Failed to create session: %s", err)
	}
	j.stdin, _ = j.ssh.session.StdinPipe()
	j.stdout, _ = j.ssh.session.StdoutPipe()
	j.stderr, _ = j.ssh.session.StderrPipe()

	j.shutdown = make(chan bool, 1)
	j.attachWg = sync.WaitGroup{}
	go j.publisher.Attach(j.stdout, j.stderr, j.shutdown, j.attachWg)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request PTY
	if err := j.ssh.session.RequestPty("xterm", 100, 100, modes); err != nil {
		j.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := j.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	j.connOptions = options
	fmt.Println("Secure shell session created.")
	// brute set terminal length 0. Could be configured to detect type and send the correct line.
	fmt.Println("Setting terminal length.")
	j.stdin.Write([]byte("terminal length 0\r"))
	j.stdin.Write([]byte("set length 0\r"))
	j.ready = true
	return nil
}

func (j *juniper) connectTelnet(options schema.ConnectOptions) (err error) {
	j.connOptions = options
	if j.connOptions.Port == 0 {
		j.connOptions.Port = 23
	}
	// connect to the host
	host := fmt.Sprintf("%v:%v", j.connOptions.Host, options.Port)

	j.shutdown = make(chan bool, 1)
	j.attachWg = sync.WaitGroup{}

	j.telnet.conn, err = gote.Dial("tcp", host)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("TCP Connected, trying to login.")

	j.stdout = j.telnet.conn
	j.stdin = j.telnet.conn

	go j.publisher.Attach(j.stdout, nil, j.shutdown, j.attachWg)

	ready, err := j.loginTelnet(options.Username, options.Password)
	if err != nil {
		fmt.Println("unable to login to telnet using username/password combination.")
		return err
	}

	if !ready {
		fmt.Println("Unable to login to telnet, device is not ready.")
		return errors.New("Device not ready.")
	}

	fmt.Println("Logged in to telnet. Connection ready.")
	// brute set terminal length 0. Could be configured to detect type and send the correct line.
	fmt.Println("Setting terminal length.")
	j.stdin.Write([]byte("terminal length 0\r"))
	j.stdin.Write([]byte("set length 0\r"))
	j.ready = true
	// need to login now
	return nil
}

func (j *juniper) loginTelnet(username, password string) (bool, error) {
	// detect "Login:" prompt
	lr, err := regexp.Compile(`.*?[Ll]ogin:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = j.writeExpectTimeout("", lr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	// detect "Password:" prompt
	pr, err := regexp.Compile(`^.*?[Pp]assword:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = j.writeExpectTimeout(username, pr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	_, err = j.writeExpectTimeout(password, j.prompt, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (j *juniper) Disconnect() bool {
	if j.connOptions.Method == SSH {
		j.ssh.session.Close()
	}
	if j.connOptions.Method == Telnet {
		// write "exit" to the stream?
		j.stdin.Write([]byte("exit\r"))
	}
	j.stdin.Close()
	j.shutdown <- true
	j.attachWg.Wait()
	return true
}

func (j *juniper) Enable(password string) (err error) {
	return nil
}

func (j *juniper) Expect(expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	return j.WriteExpectTimeout("", expectation, timeout)
}

func (j *juniper) Write(command string, newline bool) (int, error) {
	if newline {
		command += "\r"
	}
	return j.stdin.Write([]byte(command))
}

func (j *juniper) WriteCapture(command string) (result []string, err error) {
	return j.WriteExpectTimeout(command, j.prompt, j.timeout)
}

func (j *juniper) WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error) {
	return j.WriteExpectTimeout(command, expectation, j.timeout)
}

func (j *juniper) WriteExpectTimeout(command string, expectation *regexp.Regexp,
	timeout time.Duration) (result []string, err error) {
	if !j.ready {
		return result, errors.New("Device not ready to send another write command that requires capturing.")
	}
	j.ready = false
	return j.writeExpectTimeout(command, expectation, timeout)
}

func (j *juniper) writeExpectTimeout(command string, expectation *regexp.Regexp,
	timeout time.Duration) (result []string, err error) {
	events := make(chan schema.MessageEvent, 20)
	id := j.publisher.Subscribe(events)

	defer func() {
		j.ready = true
		j.publisher.Unsubscribe(id)
	}()

	if len(command) > 0 {
		// write the command
		fmt.Println("Writing command: ", string(command))
		_, err = j.Write(command, true)
		if err != nil {
			// Unable to write command
			return []string{}, err
		}
	}

	return j.expect(events, expectation, timeout)
}

func (j *juniper) expect(events chan schema.MessageEvent, expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	// Create the timeout timer using this device types default
	timer := time.NewTimer(timeout)
	for {
		select {
		case event := <-events:
			if event.Dir == schema.Stdout {
				result = append(result, event.Message)
				if found := j.match(event.Message, expectation); found {
					fmt.Println("Expectation matched.")
					return result, nil
				}
			}
			if event.Dir == schema.Stderr {
				result = append(result, event.Message)
			}
			timer.Reset(timeout)
			j.handleContinuation(event.Message)
		case <-timer.C:
			return result, errors.New("Command timeout reached without detecting expectation.")
		default:
			time.Sleep(time.Duration(20) * time.Millisecond)
		}
	}
}

func (j *juniper) match(line string, reg *regexp.Regexp) bool {
	return reg.Find([]byte(line)) != nil
}

// Options should return the connection options used for the current connection, if any
func (j *juniper) Options() schema.ConnectOptions {
	runtime.Gosched()
	return j.connOptions
}

func (j *juniper) handleContinuation(line string) {
	for _, con := range j.continuation {
		if matched := con.Find([]byte(line)); matched != nil {
			fmt.Println("Found continuation request.")
			j.Write(" ", true)
		}
	}
}
