package events

import (
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
)

func startFileWatch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// build and publish message
				message := &Message{ "name": event.Name, "op": event.Op }
				bus.Publish("file:" + event.Name, message)
				log.Debugf("File event: %v", message)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("Error: %s", err)
			}
		}
	}()

	// subscribe for watch requests, this acts as a service which allows
	// uncoupled but interested methods to watch files by using the message bus
	err = bus.Subscribe("file:watch", func(message *Message){
		if value, ok := (*message)["path"]; ok {
			if path, ok := value.(string); ok {
				// ensure removed before new watch
				_ = watcher.Remove(path)
				// start watching
				err := watcher.Add(path)
				if err != nil {
					log.Errorf("Failed to watch path: %s", err)
				} else {
					log.Debugf("Watching file: %s", path)
				}
			}
		}
	})
	if err != nil {
		log.Errorf("Failed to listen for new file watch events: %s", err)
	}
	<-done
}

// start the file watch
// todo: fix this, this is really really really bad practice
func init() {
	go startFileWatch()
}