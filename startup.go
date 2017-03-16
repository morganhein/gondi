package gondi

import (
	"github.com/morganhein/gondi/devices"
	"github.com/morganhein/gondi/dispatch"
)

type Manager struct {
	devices      map[string]devices.Device
	dispatchQuit chan bool
}

func NewG() *Manager {
	g := &Manager{
		devices:      make(map[string]devices.Device),
		dispatchQuit: make(chan bool, 1),
	}
	dispatch.Start(g.dispatchQuit)
	return g
}
