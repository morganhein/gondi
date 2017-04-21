package transport

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/morganhein/go-telnet"
	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

type ciscoios struct {
	base
}

func (c *ciscoios) Initialize() error {
	c.events = make(chan schema.MessageEvent, 20)
	c.publisher = pubsub.New(c, c.events)
	c.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`^.*?--More-- $`} {
		if re, err := regexp.Compile(next); err == nil {
			c.continuation = append(c.continuation, re)
		}
	}
	c.ready = false
	c.timeout = time.Duration(10) * time.Second
	return nil
}

func (c *ciscoios) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		if err := c.connectSsh(options); err != nil {
			return err
		}
		// brute set terminal length 0. Could be configured to detect type and send the correct line.
		log.Debug("Setting terminal length.")
		c.stdin.Write([]byte("terminal length 0\r"))
		return nil
	}
	if method == Telnet {
		options.Method = Telnet
		if err := c.connectTelnet(options); err != nil {
			return err
		}
		// brute set terminal length 0. Could be configured to detect type and send the correct line.
		log.Debug("Setting terminal length.")
		c.stdin.Write([]byte("terminal length 0\r"))
		return nil
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (c *ciscoios) connectSsh(options schema.ConnectOptions) error {
	c.ssh.Config = CreateSSHConfig(options)
	//c.ssh.Config.Ciphers = []string{
	//	"aes128-cbc",
	//	"aes256-cbc",
	//	"aes128-ctr",
	//	"aes192-ctr",
	//	"aes256-ctr",
	//	"aes128-gcm@openssh.com",
	//	"arcfour256",
	//	"arcfour128",
	//}
	c.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	log.Info(c.ssh.Config.Ciphers)
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
	if err := c.ssh.session.RequestPty("xterm", 100, 100, modes); err != nil {
		c.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := c.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	c.connOptions = options
	log.Info("SSH session created.")
	c.ready = true
	return nil
}

func (c *ciscoios) connectTelnet(options schema.ConnectOptions) (err error) {
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
		log.Info(err)
		return err
	}

	log.Debug("TCP Connected, trying to login.")

	c.stdout = c.telnet.conn
	c.stdin = c.telnet.conn

	go c.publisher.Attach(c.stdout, nil, c.shutdown, c.attachWg)

	ready, err := c.loginTelnet(options.Username, options.Password)
	if err != nil {
		log.Warningf("Unable to login to telnet using username/password combination.")
		return err
	}

	if !ready {
		log.Warning("Unable to login to telnet, device is not ready.")
		return errors.New("Device not ready.")
	}

	log.Info("Telnet session created.")
	c.ready = true
	// need to login now
	return nil
}

func (c *ciscoios) loginTelnet(username, password string) (bool, error) {
	// detect "Login:" prompt
	lr, err := regexp.Compile(`.*?[Uu]sername:? *?$`)
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
