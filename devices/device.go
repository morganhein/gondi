package devices

import (
	"bufio"
	"fmt"
	"io"

	"github.com/morganhein/gondi/schema"
)

const (
	Cisco = iota
)

const (
	SSH = iota
	Telnet
)

func New(deviceType byte) schema.Device {
	switch deviceType {
	case Cisco:
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	default:
		d := &cisco{}
		err := d.Initialize()
		if err != nil {
			return nil
		}
		return d
	}
}

func dispatch(stdin io.WriteCloser, stdout, stderr io.Reader, output chan schema.MessageEvent, shutdown chan bool) {
	for {
		select {
		case <-shutdown:
			break
		}
	}
}

func dispatchReader(io io.Reader, output chan schema.MessageEvent, shutdown chan bool) {
	scanner := bufio.NewReader(io)
	for {
		select {
		case <-shutdown:
			break
		}
		if scanner.Buffered() > 0 {
			line, err := scanner.ReadString('\n')
			if err != nil {
				fmt.Printf("Encountered error when trying to read the next line: %v", err.Error())
				continue
			}
			output <- line
		}
	}
}
