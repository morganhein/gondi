package gondi

import (
	"errors"

	"github.com/morganhein/gondi/devices"
	"github.com/morganhein/gondi/schema"
)

type Manager struct {
	devices      map[string]schema.Device
	dispatchQuit chan bool
}

func NewG() *Manager {
	g := &Manager{
		devices:      make(map[string]schema.Device),
		dispatchQuit: make(chan bool, 1),
	}
	return g
}

// Connect tries to connect to the given device using the proposed method. It does not handle trying to connect
// using other methods if the primary one fails, that should be handled upstream if there is an error.
func (m *Manager) Connect(deviceType byte, id string, method byte, options schema.ConnectOptions) (schema.Device, error) {
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

func (m *Manager) GetDevice(id string) (device schema.Device, err error) {
	return m.devices[id], nil
}

func (m *Manager) Shutdown() error {
	for _, d := range m.devices {
		d.Disconnect()
	}
	return nil
}
