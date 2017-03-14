package devices

import (
	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/schema"
)

const (
	Cisco = iota
)

func New(options DeviceOptions) Device {
	switch options.DeviceType {
	case Cisco:
		return &cisco{
			prompt:       `> *\$|# *\$|\$ *$`,
			continuation: []string{"--more--"},
		}
	default:
		return &cisco{
			prompt:       `> *\$|# *\$|\$ *$`,
			continuation: []string{"--more--"},
		}
	}
}
