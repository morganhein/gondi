package transport

import (
	"github.com/morganhein/gondi/logger"
	"github.com/morganhein/gondi/schema"
)

const (
	Cisco schema.DeviceType = iota
	CiscoXE
	CiscoXR
	Casa
	Juniper
	Adva
	IronFoundry
)

const (
	SSH schema.ConnectionMethod = iota
	Telnet
)

var log schema.Logger

func init() {
	log = logger.Log
}

func New(deviceType schema.DeviceType) schema.Device {
	log := logger.Log
	switch deviceType {
	case Cisco:
		log.Debug("Creating a new Cisco device.")
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	case Casa:
		log.Debug("Creating a new Casa device.")
		d := &casa{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	case Juniper:
		log.Debug("Creating a new Juniper device.")
		d := &juniper{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		log.Debug("Unknown device requested, making a Cisco device.")
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}
