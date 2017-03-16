package dispatch

import "sort"

type Dispatcher struct {
	input       chan Event
	subscribers map[int]chan Event
}

type EventType int

const (
	Stdin  EventType = iota
	Stderr EventType = iota
	Stdout EventType = iota
)

type singleton struct {
	subscribers map[int]chan Event
}

var dist singleton

func init() {
	dist = singleton{
		subscribers: make(map[int]chan Event, 2),
	}
}

type Event struct {
	source  string
	message string
	dir     EventType
}

func New(input chan Event) *Dispatcher {
	return &Dispatcher{
		input:       input,
		subscribers: make(map[int]chan Event, 2),
	}
}

//Subscribe adds a listener to this dispatcher.
func (d *Dispatcher) Subscribe(sub chan Event) (id int) {
	//create a slice of the keys, sorted
	keys := make([]int, len(d.subscribers))
	i := 0
	for k := range d.subscribers {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	//Add the sub to the map with the next id in order
	//todo: this could create an overflow if too many subscribers are attached to a single device
	next := keys[len(keys)-1] + 1
	d.subscribers[next] = sub
	return next
}

func (d *Dispatcher) Unsubscribe(id int) {
	if _, ok := d.subscribers[id]; ok {
		delete(d.subscribers, id)
	}
}

func (d *Dispatcher) Start(quit chan bool) {
	go func() {
		for {
			select {
			case <-quit:
				break
			case line := <-d.input:
				// Send to the creator of this dispatcher
				for _, sub := range d.subscribers {
					if len(sub) < 20 {
						sub <- line
					}
				}
				// Send to the distribution/plugins subscribers
				for _, sub := range dist.subscribers {
					if len(sub) < 20 {
						sub <- line
					}
				}
			}
		}
	}()
}

func Subscribe(sub chan Event) (id int) {
	//create a slice of the keys, sorted
	keys := make([]int, len(dist.subscribers))
	i := 0
	for k := range dist.subscribers {
		keys[i] = k
		i++
	}
	sort.Ints(keys)
	//Add the sub to the map with the next id in order
	//todo: this could create an overflow if too many subscribers are attached to a single device
	next := keys[len(keys)-1] + 1
	dist.subscribers[next] = sub
	return next
}
