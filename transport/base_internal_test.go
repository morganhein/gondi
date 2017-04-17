package transport

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/morganhein/gondi/schema"
	"github.com/stretchr/testify/assert"
)

func TestCasa_expect(t *testing.T) {
	c := &casa{}
	err := c.Initialize()

	assert.NoError(t, err)

	events := make(chan schema.MessageEvent)
	lr, _ := regexp.Compile(`^[Ll]ogin:? *?$`)

	wgClient := &sync.WaitGroup{}
	wgClient.Add(1)

	go func() {
		res, err := c.expect(events, lr, time.Duration(10)*time.Second)
		assert.NoError(t, err)
		assert.Equal(t, []string{"Login:"}, res)
		wgClient.Done()
	}()

	e := schema.MessageEvent{
		Dir:     schema.Stdout,
		Message: "Login:",
	}
	events <- e
	wgClient.Wait()
}
