package gondi

import (
	"errors"

	"github.com/morganhein/gondi/logger"
	"github.com/morganhein/gondi/schema"
	"github.com/morganhein/gondi/transport"
)

type Manager struct {
	devices      map[string]schema.Device
	dispatchQuit chan bool
	log          schema.Logger
}

func NewG() *Manager {
	g := &Manager{
		devices:      make(map[string]schema.Device),
		dispatchQuit: make(chan bool, 1),
		log:          logger.Log,
	}
	return g
}

// Connect tries to connect to the given device using the proposed method. It does not handle trying to connect
// using other methods if the primary one fails, that should be handled upstream if there is an error.
func (m *Manager) Connect(deviceType schema.DeviceType, id string, method schema.ConnectionMethod,
	options schema.ConnectOptions) (schema.Device, error) {
	device := transport.New(deviceType)
	m.log.Info("Trying to connect from Manager.")

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
		_ = d.Disconnect()
	}
	return nil
}
