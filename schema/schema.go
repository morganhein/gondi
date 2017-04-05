package schema

import (
	"regexp"
	"time"
)

type EventType int
type DeviceType int
type ConnectionMethod int

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
	Method         ConnectionMethod
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

type Configuration interface {
	//Enable tries to enter "enable mode" on the device
	Enable(password string) (err error)
	Show(option string) (response string, err error)
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
