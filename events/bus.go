package events

import (
	log "github.com/sirupsen/logrus"
	"github.com/vardius/message-bus"
)

type Message map[string]interface{}

type Listener func(message *Message)

type Handle struct {
	topic string
	listener Listener
}

var bus = messagebus.New(100)

func Send(topic string, message *Message) {
	bus.Publish(topic, message)
}

func Listen(topic string, listener Listener) *Handle {
	err := bus.Subscribe(topic, listener)
	if err != nil {
		log.Errorf("Error subscribing to topic <%s>: %s", topic, err)
		return nil
	}
	return &Handle{
		topic: topic,
		listener: listener,
	}
}

func (handle *Handle) Close() {
	err := bus.Unsubscribe(handle.topic, handle.listener)
	if err != nil {
		log.Errorf("Could not close handle for topic %s: %s", handle.topic, err)
	} else {
		log.Debugf("Removed handler for topic: %s", handle.topic)
	}
}