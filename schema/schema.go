package schema

import (
	"regexp"
	"time"
)

type EventType int
type DeviceType int
type ConnectionMethod int
type TransferMethod int

const (
	SCP TransferMethod = iota
	FTP
	TFTP
)

const (
	Stdin  EventType = iota
	Stderr EventType = iota
	Stdout EventType = iota
)

type MessageEvent struct {
	Source  Device
	Message string
	Dir     EventType
	Time    time.Time
}

type ConnectOptions struct {
	Host           string
	Port           int
	Username       string
	Password       string
	EnablePassword string
	Cert           string
	Method         ConnectionMethod // the method that this connection was successful with? not sure
}

type TransferOptions struct {
	Host     string
	Port     int
	Username string
	Password string
	Cert     string
	Method   TransferMethod
}

type Device interface {
	//Initialize sets up the device. Must be called prior to using the device
	Initialize() error
	//SupportedMethods is a list of supported connection methods, ie SSH, Telnet, CarrierPigeon
	SupportedMethods() []ConnectionMethod
	//Connect tries to connect using the devices connection options, and optional arguments
	Connect(method ConnectionMethod, options ConnectOptions, args ...string) error
	//Disconnect closes the sessions and removes all references to it in the devices module
	Disconnect() bool
	//Expect waits for timeout duration and tries to match expectation.
	Expect(expectation *regexp.Regexp, timeout time.Duration) (result []string, err error)
	//Write sends the command on the wire, optionally with a return character at the end
	Write(command string, newline bool) (sent int, err error)
	//WriteExpect writes to the device, waits for the expectation, and returns the captured text
	WriteExpect(command string, expectation *regexp.Regexp) (result []string, err error)
	//WriteCapture is a shortcut for WriteExpect(command, device.prompt)
	WriteCapture(command string) (result []string, err error)
	//WriteExpectTimeout writes the command to the device, waiting timeout duration for the expectation to match,
	//returning the captured text between command and expectation, or an error if incomplete
	WriteExpectTimeout(command string, expectation *regexp.Regexp, timeout time.Duration) (result []string, err error)
	//Options returns the connection options used for this device
	Options() ConnectOptions
}

type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	Critical(args ...interface{})
	Criticalf(format string, args ...interface{})
}

type Interaction interface {
	Device
	//Enable tries to enter "enable mode" on the device.
	Enable() (err error)
	//Enabled returns true if in enabled mode.
	Enabled() bool
	//ShowCurrent shows the running configuration. This can be either a config
	//retrieved by using the SaveConfig command, or by a capture of the "show run" or
	//analogous command.
	//The Cached flag determines whether to retrieve a new version of the config, or
	//to use a previously and current (probably) config
	ShowConfig(cached bool) (response string, err error)
	//SaveConfig saves the local config to a remote server
	SaveConfig(options TransferOptions, file string) (err error)
	//LoadConfig retrieves a config from a remote server and loads it
	LoadConfig(options TransferOptions, file string) (err error)
	//Ping sends a ping to the destination IP
	Ping(ip string, tries, timeout int) (result string, err error)

	//Possible other methods
	//AddVlan tries to add the specified VLAN
	AddVlan(identifier, comments string) (err error)
	//RemVlan tries to remove the specified VLAN
	RemVlan(identifier string) (err error)
	//AddToVlan tries to add the specified interface to a VLAN
	AddToVlan(identifier, vlan string) (err error)
	//RemFromVlan tries to remove the specified interface from a VLAN
	RemFromVlan(identifier, vlan string) (err error)
	//Motd tries to set the motd/login banner of the device
	Motd(motd string) (err error)
	//Configure tries to enter "configure" mode, if available
	Configure() (err error)
	//What to call the "Configured" method? To detect if in configuration mode?
	//InterfaceUp tries to bring up the specified interface
	InterfaceUp(identifier string) (err error)
	//InterfaceDown tries to shutdown the specified interface
	InterfaceDown(identifier string) (err error)
	//SaveCurrent saves the running configuration to memory
	SaveCurrent() (err error)
}
