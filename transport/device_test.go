package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDevice(t *testing.T) {
	d := New(Cisco)
	assert.IsType(t, &ciscoios{}, d)

	d = New(CiscoXR)
	assert.IsType(t, &ciscoxr{}, d)

	d = New(Casa)
	assert.IsType(t, &casa{}, d)

	d = New(Foundry)
	assert.IsType(t, &foundry{}, d)

	d = New(Juniper)
	assert.IsType(t, &juniper{}, d)
}
