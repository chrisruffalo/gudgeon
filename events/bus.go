package events

import (
	"github.com/asaskevich/EventBus"
	log "github.com/sirupsen/logrus"
)

type Message map[string]interface{}

type Listener func(message *Message)

type Handle struct {
	topic string
	listener Listener
}

var bus = EventBus.New()

func Send(topic string, message *Message) {
	bus.Publish(topic, message)
	log.Tracef("Sent '%s' message %v", topic, message)
}

func Listen(topic string, listener Listener) *Handle {
	err := bus.SubscribeAsync(topic, listener, false)
	if err != nil {
		log.Errorf("Error subscribing to topic '%s': %s", topic, err)
		return nil
	} else {
		log.Debugf("Listening to topic '%s'", topic)
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