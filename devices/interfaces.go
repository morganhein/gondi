package devices

import (
	"github.com/emersion/go-imap/commands"
	"io"
)

type Device interface {
	SupportedMethods() []byte
	Connect(method byte, options ConnectOptions, args ...string) error
	Disconnect()
	Write(command string) (err error) //returns when ready for the next command
	WriteExpect(command, expectation string) (result []string, err error)
	WriteCapture(command string) (result []string, err error) // shortcut for WriteExpect(cmd, prompt)
	Options() DeviceOptions
}
