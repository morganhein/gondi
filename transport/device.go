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
	"github.com/morganhein/gondi/logger"
	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

const (
	Cisco schema.DeviceType = iota
	CiscoXE
	CiscoXR
	Casa
	Juniper
	Adva
	IronFoundry
)

const (
	SSH schema.ConnectionMethod = iota
	Telnet
)

var log schema.Logger

func init() {
	log = logger.Log
}

func New(deviceType schema.DeviceType) schema.Device {
	log := logger.Log
	switch deviceType {
	case CiscoXR:
		log.Debug("Creating a new Cisco device.")
		d := &ciscoxr{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	case Casa:
		log.Debug("Creating a new Casa device.")
		d := &casa{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	case Juniper:
		log.Debug("Creating a new Juniper device.")
		d := &juniper{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		log.Debug("Unknown device requested, making a base device.")
		d := &base{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}

type base struct {
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

func (b base) Initialize() error {
	b.events = make(chan schema.MessageEvent, 20)
	b.publisher = pubsub.New(b, b.events)
	b.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`:\r$`, `:\x1B\[K$`} {
		if re, err := regexp.Compile(next); err == nil {
			b.continuation = append(b.continuation, re)
		}
	}
	b.ready = false
	b.timeout = time.Duration(30) * time.Second
	return nil
}

func (b base) SupportedMethods() []schema.ConnectionMethod {
	return []schema.ConnectionMethod{SSH, Telnet}
}

func (b base) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		log.Debug("Casa: connecting via SSH.")
		return b.connectSsh(options)
	}
	if method == Telnet {
		options.Method = Telnet
		log.Debug("Casa: connecting via Telnet.")
		return b.connectTelnet(options)
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (b base) connectSsh(options schema.ConnectOptions) error {
	b.ssh.Config = CreateSSHConfig(options)
	b.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	log.Debug("Dialing ", host)
	conn, err := ssh.Dial("tcp", host, b.ssh.Config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	b.ssh.connection = conn
	b.ssh.session, err = b.ssh.connection.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}
	b.stdin, _ = b.ssh.session.StdinPipe()
	b.stdout, _ = b.ssh.session.StdoutPipe()
	b.stderr, _ = b.ssh.session.StderrPipe()

	b.shutdown = make(chan bool, 1)
	b.attachWg = sync.WaitGroup{}
	go b.publisher.Attach(b.stdout, b.stderr, b.shutdown, b.attachWg)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request PTY
	if err := b.ssh.session.RequestPty("xterm", 0, 80, modes); err != nil {
		b.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := b.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	b.connOptions = options
	log.Info("SSH session created.")
	b.ready = true
	return nil
}

func (b base) connectTelnet(options schema.ConnectOptions) (err error) {
	b.connOptions = options
	if b.connOptions.Port == 0 {
		b.connOptions.Port = 23
	}
	// connect to the host
	host := fmt.Sprintf("%v:%v", b.connOptions.Host, options.Port)

	b.shutdown = make(chan bool, 1)
	b.attachWg = sync.WaitGroup{}

	b.telnet.conn, err = gote.Dial("tcp", host)
	if err != nil {
		log.Info(err)
		return err
	}

	log.Debug("TCP Connected, trying to login.")

	b.stdout = b.telnet.conn
	b.stdin = b.telnet.conn

	go b.publisher.Attach(b.stdout, nil, b.shutdown, b.attachWg)

	ready, err := b.loginTelnet(options.Username, options.Password)
	if err != nil {
		log.Warningf("Unable to login to telnet using username/password combination.")
		return err
	}

	if !ready {
		log.Warning("Unable to login to telnet, device is not ready.")
		return errors.New("Device not ready.")
	}

	log.Info("Telnet session created.")
	b.ready = true
	// need to login now
	return nil
}

func (b base) loginTelnet(username, password string) (bool, error) {
	// detect "Login:" prompt
	lr, err := regexp.Compile(`.*?[Ll]ogin:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = b.writeExpectTimeout("", lr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	// detect "Password:" prompt
	pr, err := regexp.Compile(`^.*?[Pp]assword:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = b.writeExpectTimeout(username, pr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	_, err = b.writeExpectTimeout(password, b.prompt, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (b base) Disconnect() bool {
	if b.connOptions.Method == SSH {
		b.ssh.session.Close()
	}
	if b.connOptions.Method == Telnet {
		// write "exit" to the stream?
		b.stdin.Write([]byte("exit\r"))
	}
	b.stdin.Close()
	b.shutdown <- true
	b.attachWg.Wait()
	return true
}

func (b base) Expect(expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	return b.WriteExpectTimeout("", expectation, timeout)
}

func (b base) Write(command string, newline bool) (sent int, err error) {
	if newline {
		command += "\r"
	}
	return b.stdin.Write([]byte(command))
}

func (b base) WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error) {
	return b.WriteExpectTimeout(command, expectation, b.timeout)
}

func (b base) WriteCapture(command string) (result []string, err error) {
	return b.WriteExpectTimeout(command, b.prompt, b.timeout)
}

func (b base) WriteExpectTimeout(command string, expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	if !b.ready {
		return result, errors.New("Device not ready to send another write command that requires capturing.")
	}
	b.ready = false
	return b.writeExpectTimeout(command, expectation, timeout)
}

func (b base) writeExpectTimeout(command string, expectation *regexp.Regexp,
	timeout time.Duration) (result []string, err error) {
	events := make(chan schema.MessageEvent, 20)
	id := b.publisher.Subscribe(events)

	defer func() {
		b.ready = true
		log.Debug("Defer unsubscribe being called.")
		b.publisher.Unsubscribe(id)
	}()

	if len(command) > 0 {
		// write the command
		log.Debug("Writing command: ", string(command))
		_, err = b.Write(command, true)
		if err != nil {
			// Unable to write command
			return []string{}, err
		}
	}

	return b.expect(events, expectation, timeout)
}

func (b base) expect(events chan schema.MessageEvent, expectation *regexp.Regexp, timeout time.Duration) (result []string, err error) {
	// Create the timeout timer using this device types default
	timer := time.NewTimer(timeout)
	for {
		select {
		case event := <-events:
			if event.Dir == schema.Stdout {
				result = append(result, event.Message)
				if found := b.match(event.Message, expectation); found {
					log.Debug("Expectation matched.")
					return result, nil
				}
			}
			if event.Dir == schema.Stderr {
				result = append(result, event.Message)
			}
			timer.Reset(timeout)
			b.handleContinuation(event.Message)
		case <-timer.C:
			return result, errors.New("Command timeout reached without detecting expectation.")
		default:
			time.Sleep(time.Duration(20) * time.Millisecond)
		}
	}
}

func (b base) Options() schema.ConnectOptions {
	runtime.Gosched()
	return b.connOptions
}

func (b base) match(line string, reg *regexp.Regexp) bool {
	return reg.Find([]byte(line)) != nil
}

func (b base) handleContinuation(line string) {
	for _, con := range b.continuation {
		if matched := con.Find([]byte(line)); matched != nil {
			log.Debug("Found continuation request.", string(matched))
			b.Write(" ", true)
		}
	}
}
