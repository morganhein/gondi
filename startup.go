package gondi

import (
	"errors"
	"github.com/morganhein/gondi/devices"
)

type Manager struct {
	devices map[string]devices.Device
}

func NewG() *Manager {
	g := &Manager{
		devices: make(map[string]devices.Device),
	}
	return g
}

// Connect tries to connect to the given device using the proposed method. It does not handle trying to connect
// using other methods if the primary one fails, that should be handled upstream if there is an error.
func (m *Manager) Connect(deviceType byte, id string, method byte, options devices.ConnectOptions) (devices.Device, error) {
	device := devices.New(deviceType)

	for _, supported := range device.SupportedMethods() {
		if supported == method {
			m.devices[id] = device
			if err := device.Connect(method, options); err != nil {
				return nil, err
			}
			return device, nil
		}
	}

	return nil, errors.New("Device does not support the method requested.")
}

func (m *Manager) GetDevice(id string) (device devices.Device, err error) {
	return m.devices[id], nil
}
