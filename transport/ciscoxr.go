package transport

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

type ciscoxr struct {
	base
}

func (c *ciscoxr) Initialize() error {
	c.events = make(chan schema.MessageEvent, 20)
	c.publisher = pubsub.New(c, c.events)
	c.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`^.*?--More-- $`} {
		if re, err := regexp.Compile(next); err == nil {
			c.continuation = append(c.continuation, re)
		}
	}
	c.ready = false
	c.timeout = time.Duration(8) * time.Second
	return nil
}

func (c *ciscoxr) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		if err := c.connectSsh(options); err != nil {
			return err
		}
		// brute set terminal length 0. Could be configured to detect type and send the correct line.
		log.Debug("Setting terminal length.")
		c.stdin.Write([]byte("terminal length 0\r"))
		c.stdin.Write([]byte("set length 0\r"))
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
		c.stdin.Write([]byte("set length 0\r"))
		return nil
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (c *ciscoxr) connectSsh(options schema.ConnectOptions) error {
	c.ssh.Config = CreateSSHConfig(options)
	c.ssh.Config.Ciphers = []string{
		"aes128-cbc",
		"aes256-cbc",
		"aes128-ctr",
		"aes192-ctr",
		"aes256-ctr",
		"aes128-gcm@openssh.com",
		"arcfour256",
		"arcfour128",
	}
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
	// brute set terminal length 0. Could be configured to detect type and send the correct line.
	log.Debug("Setting terminal length.")
	c.stdin.Write([]byte("terminal length 0\r"))
	c.stdin.Write([]byte("set length 0\r"))
	c.ready = true
	return nil
}
