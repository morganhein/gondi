package devices

const (
	SSH = iota
	Telnet
)

type ConnectOptions struct {
	Host           string
	Port           int
	Username       string
	Password       string
	EnablePassword string
	Cert           string
	Method         byte
}
