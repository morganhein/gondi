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
	base
}

func (j *juniper) Initialize() error {
	j.events = make(chan schema.MessageEvent, 20)
	j.publisher = pubsub.New(j, j.events)
	j.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`:\r$`, `:\x1B\[K$`} {
		if re, err := regexp.Compile(next); err == nil {
			j.continuation = append(j.continuation, re)
		}
	}
	j.ready = false
	j.timeout = time.Duration(30) * time.Second
	return nil
}

func (j *juniper) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		if err := j.connectSsh(options); err != nil {
			return err
		}
		log.Debug("Setting terminal length.")
		j.stdin.Write([]byte("set cli screen-length 0\r"))
		return nil
	}
	if method == Telnet {
		options.Method = Telnet
		if err := j.connectTelnet(options); err != nil {
			return err
		}
		log.Debug("Setting terminal length.")
		j.stdin.Write([]byte("set cli screen-length 0\r"))
		return nil
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (j *juniper) connectSsh(options schema.ConnectOptions) error {
	j.ssh.Config = CreateSSHConfig(options)
	//todo: add a way to provide SSH keys
	j.ssh.Config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	j.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	log.Debug("Dialing ", host)
	conn, err := ssh.Dial("tcp", host, j.ssh.Config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	j.ssh.connection = conn
	j.ssh.session, err = j.ssh.connection.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
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
	if err := j.ssh.session.RequestPty("xterm", 0, 80, modes); err != nil {
		j.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := j.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	j.connOptions = options
	log.Info("SSH session created.")
	j.ready = true
	return nil
}
