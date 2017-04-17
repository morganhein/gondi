package interaction

import (
	"fmt"
	"regexp"

	"github.com/morganhein/gondi/schema"
)

type ciscoios struct {
	schema.Device
	enablePw  string
	loginUser string
	loginPw   string
}

func (c *ciscoios) Enable(password string) (err error) {
	//Check if already enabled.
	if c.Enabled() {
		return nil
	}
	pw, _ := regexp.Compile("[pP]assword:? *?")
	// try to enable, expecting the password
	_, err = c.WriteExpect("enable", pw)
	if err != nil {
		log.Warningf("Unable to enter privileged mode on device: %s", err)
		return err
	}
	// send the password to login
	_, err = c.WriteCapture(c.enablePw)
	if err != nil {
		log.Warningf("Unable to enter privileged mode on device. Entering the password failed: %s", err)
		return err
	}
	return nil
}

func (c *ciscoios) Enabled() bool {
	// send a blank enter to detect if the prompt
	resp, err := c.WriteCapture("")
	if err != nil {
		return false
	}
	for _, x := range resp {
		fmt.Println(x)
	}
	return false
}

func (c *ciscoios) Configure() (err error) {
	panic("implement me")
}

func (c *ciscoios) ShowCurrent() (response string, err error) {
	panic("implement me")
}

func (c *ciscoios) SaveCurrent() (err error) {
	panic("implement me")
}

func (c *ciscoios) Motd(motd string) (err error) {
	panic("implement me")
}

func (c *ciscoios) InterfaceUp(identifier string) (err error) {
	panic("implement me")
}

func (c *ciscoios) InterfaceDown(identifier string) (err error) {
	panic("implement me")
}

func (c *ciscoios) AddVlan(identifier, comments string) (err error) {
	panic("implement me")
}

func (c *ciscoios) RemVlan(identifier string) (err error) {
	panic("implement me")
}

func (c *ciscoios) AddToVlan(identifier, vlan string) (err error) {
	panic("implement me")
}

func (c *ciscoios) RemFromVlan(identifier, vlan string) (err error) {
	panic("implement me")
}
