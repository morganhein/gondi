package devices

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"

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
	continuation []string
	prompt       string
}

func (c *cisco) Connect(method byte, options ConnectOptions, args ...string) error {
	if method != SSH {
		return errors.New("That connection type is currently not supported for this device.")
	}
	return c.connectSsh(options)
}

// Options should return the connection options used for the current connection, if any
func (c *cisco) Options() ConnectOptions {
	return c.connOptions
}

func (c *cisco) WriteCapture(command string) (result []string, err error) {
	panic("implement me")
}

func (c *cisco) WriteExpect(command, expectation string) (result []string, err error) {
	panic("implement me")
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
	sess, err := c.connection.NewSession()
	if err != nil {
		fmt.Errorf("Failed to create session: %s", err)
	}
	c.session = sess
	c.stdin, _ = sess.StdinPipe()
	c.stdout, _ = sess.StdoutPipe()
	c.stderr, _ = sess.StderrPipe()

	//copy to stdout, stderr
	go io.Copy(os.Stdout, c.stdout)
	go io.Copy(os.Stderr, c.stderr)

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

	// start receiving
	go c.rx()
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

func (c *cisco) Write(command string) (int, error) {
	if !c.ready {
		return 0, errors.New("Device is not ready for a new command yet.")
	}
	c.ready = false
	fmt.Printf("Sending command: %v: \n", command)
	return c.stdin.Write([]byte(command + "\r"))
}

//rx is the loop that receives stdout and stderr and copies it to output
func (c *cisco) rx() error {
	inScan := bufio.NewScanner(c.stdout)
	errScan := bufio.NewScanner(c.stderr)

	for {
		select {
		case <-c.shutdown:
			c.ready = false
			return nil
		}

		//foreach line separated by newlines
		for errScan.Scan() {
			err := errScan.Text()
			return errors.New(err)
		}

		for inScan.Scan() {
			line := inScan.Text()
			//detect if it's a prompt
			if prompt := c.detectPrompts(line); prompt {
				c.ready = true
			}
		}
	}
	return nil
}

//detectPrompts looks for prompts that require interaction like '--more--' and handles them, and also
//returns true when the normal text prompt is detected
func (c *cisco) detectPrompts(line string) bool {
	for _, con := range c.continuation {
		if matched, _ := regexp.MatchString(con, line); matched {
			// send enter key, bypassing the normal Write logic
			fmt.Fprint(c.stdin, "\r")
			return false
		}
	}
	matched, _ := regexp.MatchString(c.prompt, line)
	return matched
}
