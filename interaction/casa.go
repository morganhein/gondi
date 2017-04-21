package interaction

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/morganhein/gondi/schema"
)

type casa struct {
	base
}

func (c *casa) Storage() (result int64, err error) {
	resp, err := c.WriteCapture("")
	if err != nil {
		return 0, err
	}
	if len(resp) == 0 {
		return 0, errors.New("Unable to read file structure data.")
	}
	return c.parseStorage(resp)
}

func (c *casa) parseStorage(input []string) (result int64, err error) {
	size, err := c.retrieveStorage(input)
	if err != nil {
		return 0, nil
	}
	switch size[len(size)-1] {
	case "G":
	}

	return 0, nil
}

func (c *casa) retrieveStorage(input []string) (result string, err error) {
	l := input[len(input)-1]
	if !strings.Contains(string(l), "%") {
		return "", errors.New("Unable to find storage information in returned data.")
	}
	f := strings.Fields(string(l))
	if len(f) != 6 {
		return "", errors.New(fmt.Sprintf("Unable to parse storage information: %s", f))
	}
	return f[3], nil
}

//calculateBytes turns a size string into bytes, converting from Mb and Gb
func (c *casa) calculateBytes(size string, magnitude int) (result int64, err error) {
	i, err := strconv.ParseFloat(size, 64)
	if err != nil {
		return 0, err
	}

}

func (c *casa) LoadConfig(schema.TransferOptions, string) error {
	return nil
}
