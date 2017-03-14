package devices

const (
	SSH = iota
	Telnet
)

type DeviceOptions struct {
	Prompt       string
	Continuation []string
	DeviceType   int
	Conn         ConnectOptions
}

type ConnectOptions struct {
	Host     string
	Port     int
	Username string
	Password string
	Cert     string
}
