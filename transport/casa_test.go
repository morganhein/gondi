package transport_test

import (
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/morganhein/gondi"
	"github.com/morganhein/gondi/schema"
	"github.com/morganhein/gondi/transport"
	"github.com/stretchr/testify/assert"
)

func TestCasa_LoginTelnet(t *testing.T) {
	// create TCP server
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	wgClient := &sync.WaitGroup{}
	wgClient.Add(1)

	g := gondi.NewG()
	go func() {
		_, err := g.Connect(transport.Casa, "test-cmts", transport.Telnet, schema.ConnectOptions{
			Host:           "127.0.0.1",
			Port:           3000,
			Username:       "test",
			Password:       "password",
			EnablePassword: "enable",
		})
		assert.NoError(t, err)
		wgClient.Done()
	}()

	// listen for TCP connections
	conn, err := l.Accept()
	if err != nil {
		return
	}
	conn.Write([]byte("Login:\n"))
	buf := make([]byte, 20)

	i, err := conn.Read(buf)
	assert.NoError(t, err)
	if i != 0 {
		fmt.Println("Server: Login found: ", buf)
	}
	conn.Write([]byte("Password:\n"))

	i, err = conn.Read(buf)
	assert.NoError(t, err)
	if i != 0 {
		fmt.Println("Server: password found: ", string(buf[:i]))
	}

	conn.Write([]byte("device > "))
	defer conn.Close()
	wgClient.Wait()
}

func TestCasa_Write(t *testing.T) {
	// create TCP server
	l, err := net.Listen("tcp", ":3000")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	wgClient := &sync.WaitGroup{}
	wgClient.Add(1)

	g := gondi.NewG()
	go func() {
		dev, err := g.Connect(transport.Casa, "test-cmts", transport.Telnet, schema.ConnectOptions{
			Host:           "127.0.0.1",
			Port:           3000,
			Username:       "test",
			Password:       "password",
			EnablePassword: "enable",
		})
		assert.NoError(t, err)
		dev.Write("Hello", true)
		dev.Write("Goodbye", false)
		wgClient.Done()
	}()

	// listen for TCP connections
	conn, err := l.Accept()
	if err != nil {
		return
	}
	conn.Write([]byte("Login:\n"))
	buf := make([]byte, 20)

	i, err := conn.Read(buf)
	assert.NoError(t, err)
	if i != 0 {
		fmt.Println("Server: Login found: ", string(buf[:i]))
	}
	conn.Write([]byte("Password:\n"))

	i, err = conn.Read(buf)
	assert.NoError(t, err)
	if i != 0 {
		fmt.Println("Server: password found: ", string(buf[:i]))
	}
	conn.Write([]byte("device > "))

	//expect the page-off command to turn off the more prompt
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "page-off\r", string(buf[:i]))

	//expect the "Hello" with a line feed
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Hello\r", string(buf[:i]))

	//expect the "Goodbye" without a line feed
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye", string(buf[:i]))

	defer conn.Close()
	wgClient.Wait()
}

func TestCasa_WriteExpect(t *testing.T) {

}
