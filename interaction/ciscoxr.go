package interaction

import (
	"errors"
	"fmt"
	"strings"

	"github.com/morganhein/gondi/schema"
)

type ciscoxr struct {
	base
}

func (c *ciscoxr) Storage() (result string, err error) {
	resp, err := c.WriteCapture("")
	if err != nil {
		return "", err
	}
	if len(resp) == 0 {
		return "", errors.New("Unable to read file structure data.")
	}
	l := resp[len(resp)-1]
	if !strings.Contains(l, "%") {
		return "", errors.New("Unable to find storage information in returned data.")
	}
	f := strings.Fields(l)
	if len(f) != 5 {
		return "", errors.New(fmt.Sprintf("Unable to parse storage information: %s", f))
	}
	return f[3], nil
}

func (c *ciscoxr) LoadConfig(schema.TransferOptions, string) error {
	return nil
}

func (c *ciscoxr) SaveConfig(schema.TransferOptions, string) error {
	return nil
}

func (c *ciscoxr) ShowConfig(cached bool) (response string, err error) {
	return "", nil
}
