package transport

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDevice(t *testing.T) {
	d := New(Casa)
	assert.IsType(t, &casa{}, d)

	d = New(Cisco)
	assert.IsType(t, &ciscoxr{}, d)
}
