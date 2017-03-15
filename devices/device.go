package devices

const (
	Cisco = iota
)

func New(deviceType byte) Device {
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
