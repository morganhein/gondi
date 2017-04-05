package transport_test

import (
	"fmt"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

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
		fmt.Println("Server: Login found: ", string(buf[:i]))
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

	//expect the "Hello" with a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Hello\r", string(buf[:i]))

	//expect the "Goodbye" without a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye", string(buf[:i]))

	defer conn.Close()
	wgClient.Wait()
}

func TestCasa_WriteCapture(t *testing.T) {
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
		res, err := dev.WriteCapture("Goodbye")
		assert.NoError(t, err)
		assert.Equal(t, []string{"this", "is", "a", "set", "of", "commands", "device > "}, res)
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

	//expect the "Hello" with a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye\r", string(buf[:i]))

	conn.Write([]byte("this\nis\na\nset\nof\ncommands\ndevice > "))

	defer conn.Close()
	wgClient.Wait()
}

func TestCasa_WriteExpect(t *testing.T) {
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
		exp, err := regexp.Compile("^[Ss]tuff.*$")
		assert.NoError(t, err)
		res, err := dev.WriteExpect("Goodbye", exp)
		assert.NoError(t, err)
		assert.Equal(t, []string{"this", "is", "a", "Stuff and things"}, res)
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

	//expect the "Hello" with a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye\r", string(buf[:i]))

	conn.Write([]byte("this\nis\na\nStuff and things\nset\nof\ncommands\ndevice > "))

	defer conn.Close()
	wgClient.Wait()
}

func TestCasa_WriteExpectTimeout(t *testing.T) {
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
		exp, err := regexp.Compile("^[Ss]tuff.*$")
		assert.NoError(t, err)

		// Testing a success within the timeout
		res, err := dev.WriteExpectTimeout("Goodbye", exp, time.Duration(10)*time.Second)
		assert.NoError(t, err)
		assert.Equal(t, []string{"this", "is", "a", "Stuff and things"}, res)

		// Testing a failure due to timeout
		res, err = dev.WriteExpectTimeout("Goodbye", exp, time.Duration(100)*time.Millisecond)
		assert.Error(t, err)
		assert.Equal(t, []string{"this", "is", "a"}, res)

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

	//expect the "Goodbye" with a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye\r", string(buf[:i]))
	conn.Write([]byte("this\nis\na\nStuff and things\nset\nof\ncommands\ndevice > "))

	//expect the "Goodbye" with a carriage return
	i, err = conn.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, "Goodbye\r", string(buf[:i]))
	conn.Write([]byte("this\nis\na"))
	time.Sleep(time.Duration(1) * time.Second)
	conn.Write([]byte("Stuff and things\nset\nof\ncommands\ndevice > "))

	defer conn.Close()
	wgClient.Wait()
}
