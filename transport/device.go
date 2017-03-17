package transport

import (
	"github.com/morganhein/gondi/schema"
)

const (
	Cisco = iota
)

const (
	SSH = iota
	Telnet
)

func New(deviceType byte) schema.Device {
	switch deviceType {
	case Cisco:
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}