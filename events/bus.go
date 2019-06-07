package events

import (
	log "github.com/sirupsen/logrus"
	"github.com/vardius/message-bus"
)

type Message map[string]interface{}

type Listener func(message *Message)

var bus = messagebus.New(100)

func Send(topic string, message *Message) {
	bus.Publish(topic, message)
}

func Listen(topic string, listener Listener) {
	err := bus.Subscribe(topic, listener)
	if err != nil {
		log.Errorf("Error subscribing to topic <%s>: %s", topic, err)
	}
}
