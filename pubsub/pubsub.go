package pubsub

import (
	"bufio"
	"io"
	"sort"
	"sync"
	"time"

	"github.com/morganhein/gondi/logger"
	"github.com/morganhein/gondi/schema"
)

var log schema.Logger

type Publisher struct {
	device schema.Device
	input  chan schema.MessageEvent
	s      map[int]chan schema.MessageEvent
	mut    sync.RWMutex
}

type subscriber struct {
	s   map[int]chan schema.MessageEvent
	mut sync.RWMutex
}

var sub subscriber

func init() {
	log = logger.Log
	sub = subscriber{
		s:   make(map[int]chan schema.MessageEvent, 2),
		mut: sync.RWMutex{},
	}
}

// New creates a new pubsub. This should be called from a device.
// Then Attach() can be called to begin publishing.
func New(device schema.Device, input chan schema.MessageEvent) *Publisher {
	return &Publisher{
		device: device,
		input:  input,
		s:      make(map[int]chan schema.MessageEvent, 2),
		mut:    sync.RWMutex{},
	}
}

// Subscribe adds another listener to this pubsub, messages to be passed via the channel
// The id of this subscription is returned, which may be used to unsubscribe
func (p *Publisher) Subscribe(s chan schema.MessageEvent) (id int) {
	p.mut.Lock()
	defer p.mut.Unlock()
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
	log.Debug("Subscribing from id", next)
	return next
}

func (p *Publisher) Unsubscribe(id int) {
	log.Debug("Unsubscribing from id", id)
	p.mut.Lock()
	defer p.mut.Unlock()
	if _, ok := p.s[id]; ok {
		delete(p.s, id)
	}
}

// Attach creates the listeners for stdout and stderr,
// and begins the publisher to distribute the messages to all subs.
func (p *Publisher) Attach(stdout, stderr io.Reader, shutdown chan bool, wg sync.WaitGroup) {
	//log.Info()("Device attached to publisher.")
	wg.Add(1)
	defer wg.Done()
	qstdout := make(chan bool, 1)
	qstderr := make(chan bool, 1)
	if stdout != nil {
		go attachReader(p.device, stdout, schema.Stdout, p.input, qstdout)
	}
	if stderr != nil {
		go attachReader(p.device, stderr, schema.Stderr, p.input, qstderr)
	}
	loopCancel := make(chan bool, 1)
	loopWg := sync.WaitGroup{}
	go p.start(loopCancel, loopWg)

	// wait for shutdown signal
	<-shutdown

	loopCancel <- true
	qstdout <- true
	qstderr <- true

	wg.Wait()
	log.Debug("Device un-attached.")
}

func (p *Publisher) start(shutdown chan bool, wg sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for {
		select {
		case <-shutdown:
			return
		case line := <-p.input:
			// Send to the locally subscribed listeners (probably just the device)
			p.mut.RLock()
			for _, s := range p.s {
				if len(s) < 20 {
					s <- line
				}
			}
			p.mut.RUnlock()
			sub.mut.RLock()
			// Send to the externally subscribed listeners
			for _, s := range sub.s {
				if len(s) < 20 {
					s <- line
				}
			}
			sub.mut.RUnlock()
		default:
			time.Sleep(time.Duration(30) * time.Millisecond)
		}
	}
}

func attachReader(device schema.Device, r io.Reader, t schema.EventType, output chan schema.MessageEvent, stop chan bool) {
	scanner := bufio.NewScanner(r)
	onNewline := func(data []byte, atEOF bool) (advance int, token []byte, err error) {

		for i := 0; i < len(data); i++ {
			if data[i] == '\n' || data[i] == '\r' {
				return i + 1, data[:i], nil
			}
		}
		return len(data), data, nil
	}
	scanner.Split(onNewline)
	for {
		if ok := scanner.Scan(); ok {
			line := scanner.Text()
			e := schema.MessageEvent{
				Source:  device,
				Message: line,
				Dir:     t,
				Time:    time.Now(),
			}
			output <- e
			log.Debug("Pubsub sent: ", e.Message)
		} else {
			log.Warning("Scanning stopped, this error means a memory leak may be occurring.")
		}
		select {
		case <-stop:
			log.Debug("Reader loop closing.")
			return
		default:
		}
	}
}

// Subscribe adds a listener for all dispatchers.
// This will be used for third party logging
func Subscribe(s chan schema.MessageEvent) (id int) {
	sub.mut.Lock()
	defer sub.mut.Unlock()
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
