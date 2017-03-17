package transport

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

type casa struct {
	connOptions  schema.ConnectOptions
	sshConfig    *ssh.ClientConfig
	connection   *ssh.Client
	session      *ssh.Session
	ready        bool //set to false when running a command
	stdout       io.Reader
	stdin        io.WriteCloser
	stderr       io.Reader
	shutdown     chan bool //shutdown channel for the publisher
	continuation []*regexp.Regexp
	prompt       *regexp.Regexp
	events       chan schema.MessageEvent
	publisher    *pubsub.Publisher
	timeout      int            // The default timeout for this device
	attachWg     sync.WaitGroup // The waitgroup for the publisher attachment
}

func (c *casa) Initialize() error {
	c.events = make(chan schema.MessageEvent, 20)
	c.publisher = pubsub.New(c, c.events)
	c.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{"^--more--$", ``} {
		if re, err := regexp.Compile(next); err == nil {
			c.continuation = append(c.continuation, re)
		}
	}
	c.ready = false
	c.timeout = 800
	return nil
}

func (c *casa) Connect(method byte, options schema.ConnectOptions, args ...string) error {
	if method != SSH {
		return errors.New("That connection type is currently not supported for this device.")
	}
	return c.connectSsh(options)
}

func (c *casa) SupportedMethods() []byte {
	return []byte{SSH}
}

func (c *casa) connectSsh(options schema.ConnectOptions) error {
	c.sshConfig = CreateSSHConfig(options)
	c.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	conn, err := ssh.Dial("tcp", host, c.sshConfig)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	c.connection = conn
	c.session, err = c.connection.NewSession()
	if err != nil {
		fmt.Errorf("Failed to create session: %s", err)
	}
	c.stdin, _ = c.session.StdinPipe()
	c.stdout, _ = c.session.StdoutPipe()
	c.stderr, _ = c.session.StderrPipe()

	c.shutdown = make(chan bool, 1)
	c.attachWg = sync.WaitGroup{}
	go c.publisher.Attach(c.stdout, c.stderr, c.shutdown, c.attachWg)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request PTY
	if err := c.session.RequestPty("xterm", 80, 40, modes); err != nil {
		c.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := c.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	c.connOptions = options
	fmt.Println("Secure shell session created.")
	c.ready = true
	return nil
}

func (c *casa) Disconnect() bool {
	c.stdin.Close()
	c.session.Close()
	c.shutdown <- true
	c.attachWg.Wait()
	return true
}

func (c *casa) Enable(password string) (err error) {
	return nil
}

func (c *casa) Write(command string, newline bool) (int, error) {
	if newline {
		command += "\r"
	}
	return c.stdin.Write([]byte(command))
}

func (c *casa) WriteCapture(command string) (result []string, err error) {
	return c.WriteExpect(command, c.prompt)
}

func (c *casa) WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error) {
	if !c.ready {
		return result, errors.New("Device not ready to send another write command that requires capturing.")
	}

	c.ready = false

	events := make(chan schema.MessageEvent, 20)
	id := c.publisher.Subscribe(events)

	defer func() {
		c.ready = true
		c.publisher.Unsubscribe(id)
	}()

	// write the command
	_, err = c.Write(command, true)
	if err != nil {
		// Unable to write command
		return []string{}, err
	}

	//create the timeout
	cancel := make(chan bool, 1)
	go func(timeout int) {
		time.Sleep(time.Duration(timeout) * time.Second)
		fmt.Println("\nTimer expired.")
		cancel <- true
	}(c.timeout)

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
			c.handleContinuation(event.Message)
		case <-cancel:
			return result, errors.New("Command timeout reached without detecting expectation.")
		}
	}
	return []string{}, errors.New("Reached end of WriteExpect without receiving line data. This error " +
		"shouldn't happen.")
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
			// send enter key, bypassing the normal Write logic
			c.stdin.Write([]byte("\r"))
		}
	}
}
