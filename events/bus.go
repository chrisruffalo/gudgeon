package events

import (
	"sync"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type event struct {
	topic   string
	message *Message
}

type bus struct {
	mtx       sync.RWMutex
	topicMap  map[string]*map[uint32]Listener
	busChan   chan *event
	closeChan chan bool
}

type Message map[string]interface{}

type Listener func(message *Message)

type Handle struct {
	topic string
	id    uint32
}

const _queueSize = 10

var ebus = &bus{
	topicMap:  make(map[string]*map[uint32]Listener),
	busChan:   make(chan *event, _queueSize),
	closeChan: make(chan bool),
}

func Start() {
	go service()
}

func service() {
	for {
		select {
		// get events from channel for incoming events
		case event := <-ebus.busChan:
			if event.topic != "" {
				var listeners *map[uint32]Listener

				ebus.mtx.RLock()
				if foundListeners, found := ebus.topicMap[event.topic]; found {
					listeners = foundListeners
				}
				ebus.mtx.RUnlock()

				if listeners != nil {
					for id, listen := range *listeners {
						log.Tracef("Dispatched event message{%v} on topic %s to listener %d", event.message, event.topic, id)
						if listen != nil {
							listen(event.message)
						}
					}
				}
			}
		case <-ebus.closeChan:
			ebus.closeChan <- true
			return
		}
	}
}

func Stop() {
	ebus.closeChan <- true
	<-ebus.closeChan
	close(ebus.closeChan)
	close(ebus.busChan)
	ebus = nil

}

func Send(topic string, message *Message) {
	if topic == "" {
		return
	}
	if ebus == nil {
		return
	}
	ebus.busChan <- &event{
		topic:   topic,
		message: message,
	}
}

func Listen(topic string, listener Listener) *Handle {
	// return basically a no-op handler
	if ebus == nil {
		return &Handle{
			topic: "",
		}
	}

	if _, found := ebus.topicMap[topic]; !found {
		ebus.mtx.Lock()
		if _, found := ebus.topicMap[topic]; !found {
			newMap := make(map[uint32]Listener)
			ebus.topicMap[topic] = &newMap
		}
		ebus.mtx.Unlock()
	}
	listeners := ebus.topicMap[topic]

	// create new id
	id := uuid.New().ID()
	ebus.mtx.Lock()
	(*listeners)[id] = listener
	ebus.mtx.Unlock()

	return &Handle{
		topic: topic,
		id:    id,
	}
}

func unsubscribe(handle *Handle) {
	if handle == nil || ebus == nil {
		return
	}

	if _, found := ebus.topicMap[handle.topic]; found {
		ebus.mtx.Lock()
		if _, found := ebus.topicMap[handle.topic]; found && ebus.topicMap[handle.topic] != nil {
			delete(*ebus.topicMap[handle.topic], handle.id)
		}
		ebus.mtx.Unlock()
	}
}

func (handle *Handle) Close() {
	if handle.topic == "" {
		return
	}
	unsubscribe(handle)
	log.Debugf("Removed handler for topic: %s", handle.topic)
}
