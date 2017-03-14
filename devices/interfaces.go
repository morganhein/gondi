package devices

type Device interface {
	//SupportedMethods is a list of supported connection methods, ie SSH, Telnet, CarrierPigeon
	SupportedMethods() []byte
	//Connect tries to connect using the devices connection options, and optional arguments
	Connect(method byte, options ConnectOptions, args ...string) error
	//Disconnect closes the sessions and removes all references to it in the devices module
	Disconnect()
	//Enable tries to enter "enable mode" on the device
	Enable(password string) (err error)
	Write(command string) (sent int, err error) //returns when ready for the next command
	// WriteExpect writes to the device, waits for the expectation, and returns the captured text
	WriteExpect(command, expectation string) (result []string, err error)
	//WriteCapture is a shortcut for WriteExpect(command, device.prompt)
	WriteCapture(command string) (result []string, err error)
	//Options returns the connection options used for this device
	Options() ConnectOptions
}
