package gondi

import (
	"fmt"
	"github.com/morganhein/gondi/devices"
)

type Manager struct {
	devices []devices.Device
}

func NewG() *Manager {
	g := &Manager{}
	return g
}

func (m *Manager) AddDevice(id string, options devices.DeviceOptions) devices.Device {
	d := newDevice(options)
	m.devices = append(m.devices, d)
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
		if err := m.connect(d, method); err == nil {
			return
		}
		// add the device to the internal inventory

	}
}

func (m *Manager) connect(d devices.Device, method byte) error {
	return d.Connect(method)
}

func (m *Manager) SendCommands(device devices.Device, commands []string) {
	for _, cmd := range commands {
		err := device.Write(cmd)
		if err != nil {
			fmt.Printf("Error sending command %s to device %s.",
				cmd, device.Options().Conn.Host)
			return
		}
	}
}
