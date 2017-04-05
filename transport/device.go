package transport

import (
	"fmt"
	"github.com/morganhein/gondi/schema"
)

const (
	Cisco schema.DeviceType = iota
	Casa
	Juniper
	Adva
)

const (
	SSH schema.ConnectionMethod = iota
	Telnet
)

func New(deviceType schema.DeviceType) schema.Device {
	switch deviceType {
	case Cisco:
		fmt.Println("Cisco device.")
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	case Casa:
		fmt.Println("Casa device.")
		d := &casa{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		fmt.Println("Cisco device.")
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}
