package devices

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"time"

	"github.com/morganhein/gondi/dispatch"
	"golang.org/x/crypto/ssh"
)

type cisco struct {
	connOptions  ConnectOptions
	sshConfig    *ssh.ClientConfig
	connection   *ssh.Client
	session      *ssh.Session
	ready        bool //set to false when running a command
	stdout       io.Reader
	stdin        io.WriteCloser
	stderr       io.Reader
	shutdown     chan bool
	continuation []*regexp.Regexp
	prompt       *regexp.Regexp
	events       chan dispatch.Event
	dispatch     *dispatch.Dispatcher
	timeout      int // The default timeout for this device
}

func (c *cisco) Initialize() error {
	c.events = make(chan dispatch.Event, 20)
	c.dispatch = dispatch.New(c.events)
	c.prompt, _ = regexp.Compile(`> *$|# *$|$ *$`)
	for _, next := range []string{"^--more--$"} {
		if re, err := regexp.Compile(next); err == nil {
			c.continuation = append(c.continuation, re)
		}
	}
	c.timeout = 8
	return nil
}

func (c *cisco) Connect(method byte, options ConnectOptions, args ...string) error {
	if method != SSH {
		return errors.New("That connection type is currently not supported for this device.")
	}
	return c.connectSsh(options)
}

func (c *cisco) SupportedMethods() []byte {
	return []byte{SSH}
}

func (c *cisco) connectSsh(options ConnectOptions) error {
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
	c.shutdown = make(chan bool, 1)

	c.connOptions = options
	fmt.Println("Secure shell session created.")
	c.ready = true
	return nil
}

func (c *cisco) Disconnect() {
	c.stdin.Close()
	c.session.Close()
}

func (c *cisco) Enable(password string) (err error) {
	return nil
}

func (c *cisco) Write(command string, newline bool) (int, error) {
	//if !c.ready {
	//	return 0, errors.New("Device is not ready for a new command yet.")
	//}
	if newline {
		command += "\r"
	}
	return c.stdin.Write([]byte(command))
}

func (c *cisco) WriteCapture(command string) (result []string, err error) {
	return c.WriteExpect(command, c.prompt)
}

func (c *cisco) WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error) {
	if !c.ready {
		return result, errors.New("Device not ready to send another write command that requires capturing.")
	}

	c.ready = false
	//c.capture <- true

	defer func() {
		//c.capture <- false
		c.ready = true
	}()

	// write the command
	_, err = c.Write(command, true)
	if err != nil {
		// Unable to write command
		return []string{}, err
	}
	inScan := bufio.NewReader(c.stdout)

	//create the timeout
	cancel := make(chan bool, 1)
	go func(timeout int) {
		time.Sleep(time.Duration(timeout) * time.Second)
		fmt.Println("\nTimer expired.")
		cancel <- true
	}(c.timeout)

	for {
		if inScan.Buffered() > 0 {
			line, err := inScan.ReadString('\n')
			if err != nil {
				fmt.Printf("Encountered error when trying to read the next line: %v", err.Error())
				continue
			}
			result = append(result, line)
			//send enter key when needing continuation
			c.handleContinuation(line)
			//detect if it's the regex we're looking for
			if found := c.match(line, expectation); found {
				return result, nil
			}
		}
		select {
		case <-cancel:
			return result, errors.New("Command timeout reached without detecting expectation.")
		}
	}
	return []string{}, errors.New("Reached end of WriteExpect without receiving line data. This error " +
		"shouldn't happen")
}

func (c *cisco) match(line string, reg *regexp.Regexp) bool {
	return reg.Find([]byte(line)) != nil
}

// Options should return the connection options used for the current connection, if any
func (c *cisco) Options() ConnectOptions {
	runtime.Gosched()
	return c.connOptions
}

func (c *cisco) handleContinuation(line string) {
	for _, con := range c.continuation {
		if matched := con.Find([]byte(line)); matched != nil {
			// send enter key, bypassing the normal Write logic
			c.stdin.Write([]byte("\r"))
		}
	}
}

//detectPrompts looks for prompts that require interaction like '--more--' and handles them, and also
//returns true when the normal text prompt is detected
func (c *cisco) detectPrompts(line string) bool {
	if matched := c.prompt.Find([]byte(line)); matched != nil {
		fmt.Println("Detected prompt.")
		return true
	}
	return false
}

func (c *cisco) io(shutdown chan bool) {
	for {
		select {
		case <-shutdown:
			break
		}
	}
}
