package devices

const (
	Cisco = iota
)

func New(deviceType byte) Device {
	switch deviceType {
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
