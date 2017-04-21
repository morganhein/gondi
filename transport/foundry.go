package transport

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"sync"
	"time"

	"github.com/morganhein/go-telnet"
	"github.com/morganhein/gondi/pubsub"
	"github.com/morganhein/gondi/schema"
	"golang.org/x/crypto/ssh"
)

//todo: strip \u0008 from stream, the foundry seems to spit out a lot of these
//todo: SSH untested

type foundry struct {
	base
}

func (f *foundry) Initialize() error {
	f.events = make(chan schema.MessageEvent, 20)
	f.publisher = pubsub.New(f, f.events)
	f.prompt, _ = regexp.Compile(`> *$|# *$|\$ *$`)
	for _, next := range []string{`^--More--,`} {
		if re, err := regexp.Compile(next); err == nil {
			f.continuation = append(f.continuation, re)
		}
	}
	f.ready = false
	f.timeout = time.Duration(30) * time.Second
	return nil
}

func (f *foundry) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		if err := f.connectSsh(options); err != nil {
			return err
		}
		log.Debug("Unable to set terminal length without enabling first.")
		//f.stdin.Write([]byte("set cli screen-length 0\r"))
		return nil
	}
	if method == Telnet {
		options.Method = Telnet
		if err := f.connectTelnet(options); err != nil {
			return err
		}
		log.Debug("Unable to set terminal length without enabling first.")
		//f.stdin.Write([]byte("set cli screen-length 0\r"))
		return nil
	}
	return errors.New("That connection type is currently not supported for this device.")
}

func (f *foundry) connectTelnet(options schema.ConnectOptions) (err error) {
	f.connOptions = options
	if f.connOptions.Port == 0 {
		f.connOptions.Port = 23
	}
	// connect to the host
	host := fmt.Sprintf("%v:%v", f.connOptions.Host, options.Port)

	f.shutdown = make(chan bool, 1)
	f.attachWg = sync.WaitGroup{}

	f.telnet.conn, err = gote.Dial("tcp", host)
	if err != nil {
		log.Info(err)
		return err
	}

	log.Debug("TCP Connected, trying to login.")

	f.stdout = f.telnet.conn
	f.stdin = f.telnet.conn

	go f.publisher.Attach(f.stdout, nil, f.shutdown, f.attachWg)

	fmt.Println("Calling login function.")
	ready, err := f.loginTelnet(options.Username, options.Password)
	if err != nil {
		log.Warningf("Unable to login to telnet using username/password combination.")
		return err
	}

	if !ready {
		log.Warning("Unable to login to telnet, device is not ready.")
		return errors.New("Device not ready.")
	}

	log.Info("Telnet session created.")
	f.ready = true
	// need to login now
	return nil
}

func (f *foundry) connectSsh(options schema.ConnectOptions) error {
	f.ssh.Config = CreateSSHConfig(options)
	//todo: add a way to provide SSH keys
	f.ssh.Config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
	f.connOptions.Method = SSH
	host := fmt.Sprint(options.Host, ":", options.Port)
	log.Debug("Dialing ", host)
	conn, err := ssh.Dial("tcp", host, f.ssh.Config)
	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}
	f.ssh.connection = conn
	f.ssh.session, err = f.ssh.connection.NewSession()
	if err != nil {
		return fmt.Errorf("Failed to create session: %s", err)
	}
	f.stdin, _ = f.ssh.session.StdinPipe()
	f.stdout, _ = f.ssh.session.StdoutPipe()
	f.stderr, _ = f.ssh.session.StderrPipe()

	f.shutdown = make(chan bool, 1)
	f.attachWg = sync.WaitGroup{}
	go f.publisher.Attach(f.stdout, f.stderr, f.shutdown, f.attachWg)

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// Request PTY
	if err := f.ssh.session.RequestPty("xterm", 0, 80, modes); err != nil {
		f.ssh.session.Close()
		return fmt.Errorf("Request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := f.ssh.session.Shell(); err != nil {
		return fmt.Errorf("Failed to start shell: %s", err)
	}
	f.connOptions = options
	log.Info("SSH session created.")
	f.ready = true
	return nil
}

func (f *foundry) loginTelnet(username, password string) (bool, error) {
	fmt.Println("Detecting login prompt.")
	// detect "Login:" prompt
	lr, err := regexp.Compile(`.*?[Ll]ogin [Nn]ame:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = f.writeExpectTimeout("", lr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	// detect "Password:" prompt
	pr, err := regexp.Compile(`^.*?[Pp]assword:? *?$`)
	if err != nil {
		return false, err
	}
	_, err = f.writeExpectTimeout(username, pr, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	_, err = f.writeExpectTimeout(password, f.prompt, time.Duration(20)*time.Second)
	if err != nil {
		return false, err
	}
	return true, nil
}
