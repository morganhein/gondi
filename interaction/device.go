package interaction

import (
	"errors"
	"regexp"

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

var log schema.Logger

func init() {
	log = logger.Log
}

func New(deviceType schema.DeviceType) schema.Interaction {
	log := logger.Log
	switch deviceType {
	case CiscoXR:
		log.Debug("Creating a new Cisco device.")
		d := &ciscoxr{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		log.Debug("Unknown device requested, making a base device.")
		d := &base{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}

type base struct {
	schema.Device
	enablePw  string
	loginUser string
	loginPw   string
}

func (b base) Enable() (err error) {
	//Check if already enabled.
	if b.Enabled() {
		return nil
	}
	pw, _ := regexp.Compile("[pP]assword:? *?")
	// try to enable, expebting the password
	_, err = b.WriteExpect("enable", pw)
	if err != nil {
		log.Warningf("Unable to enter privileged mode on device: %s", err)
		return err
	}
	// send the password to login
	_, err = b.WriteCapture(b.enablePw)
	if err != nil {
		log.Warningf("Unable to enter privileged mode on device. Entering the password failed: %s", err)
		return err
	}
	enabled := b.Enabled()
	if !enabled {
		return errors.New("Unable to enter privileged mode. Error unknown.")
	}
	return nil
}

func (b base) Enabled() bool {
	// send a blank enter to detect if the prompt ends in a # character
	resp, err := b.WriteCapture("")
	if err != nil {
		return false
	}
	// Iterate over all the returned lines.
	// We may receive more than a single line if there are alerts or
	// other information sent at the time of testing the prompt.
	// This can send false positives if any of the lines end in a #pound symbol.
	for _, line := range resp {
		if string(line[len(line)-1]) == "#" {
			log.Debugf("Enabled response: %s, last character: %s", line, string(line[len(line)-1]))
			return true
		}
	}
	return false
}

func (b base) Configure() (err error) {
	panic("implement me")
}

func (b base) ShowCurrent() (response string, err error) {
	panic("implement me")
}

func (b base) SaveCurrent() (err error) {
	panic("implement me")
}

func (b base) DeployConfig(options schema.TransferOptions, file string) (err error) {
	panic("implement me")
}

func (b base) InterfaceUp(identifier string) (err error) {
	panic("implement me")
}

func (b base) InterfaceDown(identifier string) (err error) {
	panic("implement me")
}

func (b base) Ping(ip string, tries, timeout int) (result string, err error) {
	panic("implement me")
}

func (b base) AddVlan(identifier, comments string) (err error) {
	panic("implement me")
}

func (b base) RemVlan(identifier string) (err error) {
	panic("implement me")
}

func (b base) AddToVlan(identifier, vlan string) (err error) {
	panic("implement me")
}

func (b base) RemFromVlan(identifier, vlan string) (err error) {
	panic("implement me")
}

func (b base) Motd(motd string) (err error) {
	panic("implement me")
}
