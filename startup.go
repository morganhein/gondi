package gondi

import (
	"fmt"
	"github.com/morganhein/gondi/devices"
	"github.com/morganhein/gondi/schema"
)

type Manager struct {
	devices map[devices.Device]devices.DeviceOptions
}

func New() *Manager {
	g := &Manager{}
	return g
}

func (m *Manager) AddDevice(id string, options devices.DeviceOptions) devices.Device {
	d := devices.New(options)
	m.devices[d] = options
	return d
}

func (m *Manager) GetDevice(id string) (device devices.Device, err error) {
	//if device exists return it,
	//otherwise not found error
}

func (m *Manager) Connect(id string, options devices.DeviceOptions) {
	d := m.AddDevice(id, options)
	m.ConnectDevice(d)
}

func (m *Manager) ConnectDevice(d devices.Device) {
	for _, method := range d.SupportedMethods() {
		if err := m.connect(d, method, m.devices[d].Conn); err == nil {
			return
		}
		// add the device to the internal inventory

	}
}

func (m *Manager) connect(d devices.Device, method byte, options devices.ConnectOptions) error {
	return d.Connect(method, options)
}

func (m *Manager) SendCommands(device devices.Device, commands []string) {
	for _, cmd := range commands {
		err := device.Write(cmd)
		if err != nil {
			fmt.Printf("Error sending command %s to device %s.",
				cmd, m.devices[device].Conn.Host)
			return
		}
	}
}
