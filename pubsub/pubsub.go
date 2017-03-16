package pubsub

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/morganhein/gondi/schema"
)

type Publisher struct {
	device schema.Device
	input  chan schema.MessageEvent
	s      map[int]chan schema.MessageEvent
}

type subscriber struct {
	s map[int]chan schema.MessageEvent
}

var sub subscriber

func init() {
	sub = subscriber{
		s: make(map[int]chan schema.MessageEvent, 2),
	}
}

func New(device schema.Device, input chan schema.MessageEvent) *Publisher {
	return &Publisher{
		device: device,
		input:  input,
		s:      make(map[int]chan schema.MessageEvent, 2),
	}
}

func (p *Publisher) Subscribe(s chan schema.MessageEvent) (id int) {
	next := 0
	//create a slice of the keys
	keys := make([]int, len(p.s))
	if len(keys) > 0 {
		//sort them
		i := 0
		for k := range sub.s {
			keys[i] = k
			i++
		}
		sort.Ints(keys)

		//todo: this could create an overflow if too many subscribers are attached to a single device
		next = keys[len(keys)-1] + 1
	}
	//Add the sub to the map with the next id in order
	p.s[next] = s
	return next
}

func (p *Publisher) Unsubscribe(id int) {
	if _, ok := p.s[id]; ok {
		delete(p.s, id)
	}
}

func (p *Publisher) Attach(stdout, stderr io.Reader, shutdown chan bool) {
	fmt.Println("Device attached to publisher.")
	var stdoutShutdown chan bool
	var stderrShutdown chan bool
	if stdout != nil {
		stdoutShutdown = make(chan bool, 1)
		go attachReader(p.device, stdout, schema.Stdout, p.input, stdoutShutdown)
	}
	if stderr != nil {
		stderrShutdown = make(chan bool, 1)
		go attachReader(p.device, stderr, schema.Stderr, p.input, stderrShutdown)
	}
	dispatchShutdown := make(chan bool, 1)
	go p.publish(dispatchShutdown)
	for {
		select {
		case <-shutdown:
			if stdout != nil {
				stdoutShutdown <- true
			}
			if stderr != nil {
				stderrShutdown <- true
			}
			dispatchShutdown <- true
			break
		}
	}
	fmt.Println("Device un-attached.")
}

func (p *Publisher) publish(shutdown chan bool) {
	for {
		select {
		case <-shutdown:
			break
		case line := <-p.input:
			// Send to the locally subscribed listeners (probably just the device)
			for _, s := range p.s {
				if len(s) < 20 {
					s <- line
				}
			}
			// Send to the externally subscribed listeners
			for _, s := range sub.s {
				if len(s) < 20 {
					s <- line
				}
			}
		}
	}
}

func attachReader(device schema.Device, io io.Reader, t schema.EventType, output chan schema.MessageEvent,
	shutdown chan bool) {
	fmt.Printf("Reader of type %v attached to io.\n", t)
	scanner := bufio.NewReader(io)
	for {
		select {
		case <-shutdown:
			break
		default:
			if scanner.Buffered() > 0 {
				line, err := scanner.ReadString('\n')
				if err != nil {
					fmt.Printf("Encountered an error when trying to read the next line: %v", err.Error())
					continue
				}
				e := schema.MessageEvent{
					Source:  device,
					Message: line,
					Dir:     t,
					Time:    time.Now(),
				}
				output <- e
				fmt.Println(e.Message)
			}
		}

	}
}

// Subscribe adds a listener for all dispatchers
func Subscribe(s chan schema.MessageEvent) (id int) {
	next := 0
	//create a slice of the keys
	keys := make([]int, len(sub.s))
	if len(keys) > 0 {
		//sort them
		i := 0
		for k := range sub.s {
			keys[i] = k
			i++
		}
		sort.Ints(keys)

		//todo: this could create an overflow if too many subscribers are attached to a single device
		next = keys[len(keys)-1] + 1
	}
	//Add the sub to the map with the next id in order
	sub.s[next] = s
	return next
}
