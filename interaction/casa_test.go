package interaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCasa_Storage(t *testing.T) {
	c := &casa{}
	res, err := c.parseStorage([]string{"total 482944",
		"-rw-r--r-- 1 croot root         0 Jul  5  2016 tmp-IbLeB0",
		"-rw-r--r-- 1 croot root         0 Jul  5  2016 tmp-njOHDD",
		"Filesystem            Size  Used Avail Use% Mounted on",
		"/dev/hda1             3.8G  830M  3.0G  22% /fdsk"})
	assert.NoError(t, err)
	assert.Equal(t, "3.0G", res)
}
