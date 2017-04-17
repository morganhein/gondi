package transport

import (
	"errors"
	"github.com/morganhein/gondi/schema"
)

type casa struct {
	base
}

func (c *casa) Connect(method schema.ConnectionMethod, options schema.ConnectOptions, args ...string) error {
	if method == SSH {
		options.Method = SSH
		log.Debug("Casa: connecting via SSH.")
		if err := c.connectSsh(options); err != nil {
			return err
		}
		log.Debug("Setting terminal length.")
		c.stdin.Write([]byte("page-off\r"))
		return nil
	}
	if method == Telnet {
		options.Method = Telnet
		log.Debug("Casa: connecting via Telnet.")
		if err := c.connectTelnet(options); err != nil {
			return err
		}
		log.Debug("Setting terminal length.")
		c.stdin.Write([]byte("page-off\r"))
		return nil
	}
	return errors.New("That connection type is currently not supported for this device.")
}
