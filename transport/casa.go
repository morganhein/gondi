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

type casa struct {
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

func (c *casa) Initialize() error {
	c.events = make(chan schema.MessageEvent, 20)
	c.publisher = pubsub.New(c, c.events)
	c.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`:\r$`, `:\x1B\[K$`} {
		if re, err := regexp.Compile(next); err == nil {
			c.continuation = append(c.continuation, re)
		}
	}
	c.ready = false
	c.timeout = time.Duration(8) * time.Second
	return nil
}

func (c *casa) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		fmt.Println("Casa: connecting via SSH.")
		return c.connectSsh(options)
	}
	if method == Telnet {
		options.Method = Telnet
		fmt.Println("Casa: connecting via Telnet.")
		return c.connectTelnet(options)
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (c *casa) SupportedMethods() []schema.ConnectionMethod {
	return []schema.ConnectionMethod{SSH, Telnet}
}

func (c *casa) connectSsh(options schema.ConnectOptions) error {
	c.ssh.Config = CreateSSHConfig(options)
	c.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	fmt.Println("Dialing ", host)
	conn, err := ssh.Dial("tcp", host, c.ssh.Config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	c.ssh.connection = conn
	c.ssh.session, err = c.ssh.connection.NewSession()
	if err != nil {
		fmt.Errorf("Failed to create session: %s", err)
	}
	c.stdin, _ = c.ssh.session.StdinPipe()
	c.stdout, _ = c.ssh.session.StdoutPipe()
	c.stderr, _ = c.ssh.session.StderrPipe()

	c.shutdown = make(chan bool, 1)
	c.attachWg = sync.WaitGroup{}
	go c.publisher.Attach(c.stdout, c.stderr, c.shutdown, c.attachWg)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request PTY
	if err := c.ssh.session.RequestPty("xterm", 0, 80, modes); err != nil {
		c.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := c.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	c.connOptions = options
	fmt.Println("Secure shell session created.")
	fmt.Println("Setting terminal length.")
	c.stdin.Write([]byte("page-off\r"))
	c.ready = true
	return nil
}

func (c *casa) connectTelnet(options schema.ConnectOptions) (err error) {
	c.connOptions = options
	if c.connOptions.Port == 0 {
		c.connOptions.Port = 23
	}
	// connect to the host
	host := fmt.Sprintf("%v:%v", c.connOptions.Host, options.Port)

	c.shutdown = make(chan bool, 1)
	c.attachWg = sync.WaitGroup{}

	c.telnet.conn, err = gote.Dial("tcp", host)
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("TCP Connected, trying to login.")

	c.stdout = c.telnet.conn
	c.stdin = c.telnet.conn

	go c.publisher.Attach(c.stdout, nil, c.shutdown, c.attachWg)

	ready, err := c.loginTelnet(options.Username, options.Password)
	if err != nil {
		fmt.Println("unable to login to telnet using username/password combination.")
		return err
	}

	if !ready {
		fmt.Println("Unable to login to telnet, device is not ready.")
		return errors.New("Device not ready.")
	}

	fmt.Println("Logged in to telnet. Connection ready.")
	fmt.Println("Setting terminal length.")
	c.stdin.Write([]byte("page-off\r"))
	c.ready = true
	// need to login now
	return nil
}

func (c *casa) loginTelnet(username, password string) (bool, error) {
	// detect "Login:" prompt
	lr, err := regexp.Compile(`.*?[Ll]ogin:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = c.writeExpectTimeout("", lr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	// detect "Password:" prompt
	pr, err := regexp.Compile(`^.*?[Pp]assword:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = c.writeExpectTimeout(username, pr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	_, err = c.writeExpectTimeout(password, c.prompt, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *casa) Disconnect() bool {
	if c.connOptions.Method == SSH {
		c.ssh.session.Close()
	}
	if c.connOptions.Method == Telnet {
		// write "exit" to the stream?
		c.stdin.Write([]byte("exit\r"))
	}
	c.stdin.Close()
	c.shutdown <- true
	c.attachWg.Wait()
	return true
}

func (c *casa) Enable(password string) (err error) {
	return nil
}

func (c *casa) Expect(expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	return c.WriteExpectTimeout("", expectation, timeout)
}

func (c *casa) Write(command string, newline bool) (int, error) {
	if newline {
		command += "\r"
	}
	return c.stdin.Write([]byte(command))
}

func (c *casa) WriteCapture(command string) (result []string, err error) {
	return c.WriteExpectTimeout(command, c.prompt, c.timeout)
}

func (c *casa) WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error) {
	return c.WriteExpectTimeout(command, expectation, c.timeout)
}

func (c *casa) WriteExpectTimeout(command string, expectation *regexp.Regexp,
	timeout time.Duration) (result []string, err error) {
	if !c.ready {
		return result, errors.New("Device not ready to send another write command that requires capturing.")
	}
	c.ready = false
	return c.writeExpectTimeout(command, expectation, timeout)
}

func (c *casa) writeExpectTimeout(command string, expectation *regexp.Regexp,
	timeout time.Duration) (result []string, err error) {
	events := make(chan schema.MessageEvent, 20)
	id := c.publisher.Subscribe(events)

	defer func() {
		c.ready = true
		fmt.Println("Defer unsubscribe being called.")
		c.publisher.Unsubscribe(id)
	}()

	if len(command) > 0 {
		// write the command
		fmt.Println("Writing command: ", string(command))
		_, err = c.Write(command, true)
		if err != nil {
			// Unable to write command
			return []string{}, err
		}
	}

	return c.expect(events, expectation, timeout)
}

func (c *casa) expect(events chan schema.MessageEvent, expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	// Create the timeout timer using this device types default
	timer := time.NewTimer(timeout)
	for {
		select {
		case event := <-events:
			if event.Dir == schema.Stdout {
				result = append(result, event.Message)
				if found := c.match(event.Message, expectation); found {
					fmt.Println("Expectation matched.")
					return result, nil
				}
			}
			if event.Dir == schema.Stderr {
				result = append(result, event.Message)
			}
			timer.Reset(timeout)
			c.handleContinuation(event.Message)
		case <-timer.C:
			return result, errors.New("Command timeout reached without detecting expectation.")
		default:
			time.Sleep(time.Duration(20) * time.Millisecond)
		}
	}
}

func (c *casa) match(line string, reg *regexp.Regexp) bool {
	return reg.Find([]byte(line)) != nil
}

// Options should return the connection options used for the current connection, if any
func (c *casa) Options() schema.ConnectOptions {
	runtime.Gosched()
	return c.connOptions
}

func (c *casa) handleContinuation(line string) {
	for _, con := range c.continuation {
		if matched := con.Find([]byte(line)); matched != nil {
			fmt.Println("Found continuation request.", string(matched))
			c.Write(" ", true)
		}
	}
}
