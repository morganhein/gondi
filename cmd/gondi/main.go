package main

import (
	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/schema"
)

func main() {
	g := gondi.NewG()
	g.AddDevice("test", schema.DeviceOptions{})
}
