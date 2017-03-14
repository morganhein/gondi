package devices

const (
	Cisco = iota
)

func newDevice(options DeviceOptions) Device {
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
