package events

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

// hashes of files, only propagate event if hash changes
var watchedFilesHashes = make(map[string]string)

func pathHash(fileToHash string) string {
	file, err := os.Open(fileToHash)
	if err != nil {
		return ""
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return ""
	}

	return hex.EncodeToString(hash.Sum(nil))
}

// only do this once
func StartFileWatch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("While creating watcher: %s", err)
	}

	// channel to signify end of watch
	endWatch := make(chan bool)

	go func() {
		log.Debugf("Started file change loop")
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if _, err := os.Stat(event.Name); err != nil {
					log.Debugf("Error accessing file, eating event: %s", err)
					continue
				}
				log.Tracef("file event: %s", event.Name)
				// calculate new hash
				newHash := pathHash(event.Name)
				// if the new hash was calculated it means there is a file. if the hash didn't exist previously
				// or the new hash and the old hash don't match send a message
				if oldHash, found := watchedFilesHashes[event.Name]; "" != newHash && (!found || newHash != oldHash) {
					// set changed hash
					watchedFilesHashes[event.Name] = newHash
					// build and publish message
					message := &Message{"name": event.Name, "op": event.Op}
					Send("file:"+event.Name, message)
					log.Debugf("File event: %v", message)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("Error: %s", err)
			case <- endWatch:
				endWatch <- true
				log.Debugf("File change watcher event loop closed")
				return
			}
		}
	}()

	// subscribe for watch requests, this acts as a service which allows
	// uncoupled but interested methods to watch files by using the message bus
	Listen("file:watch:start", func(message *Message) {
		if value, ok := (*message)["path"]; ok {
			if path, ok := value.(string); ok {
				// remove previous hash
				watchedFilesHashes[path] = ""
				// ensure removed before new watch
				_ = watcher.Remove(path)
				// calculate hash
				watchedFilesHashes[path] = pathHash(path)
				// start watching
				err := watcher.Add(path)
				if err != nil {
					log.Errorf("Failed to watch path: %s", err)
				} else {
					log.Tracef("Watching file: %s", path)
				}
			}
		}
	})

	// stop listening to specific files
	Listen("file:watch:end", func(message *Message) {
		if value, ok := (*message)["path"]; ok {
			if path, ok := value.(string); ok {
				// remove previous hash
				watchedFilesHashes[path] = ""
				// ensure removed before new watch
				_ = watcher.Remove(path)
			}
		}
	})

	// clear all watches
	Listen("file:watch:clear", func(message *Message) {
		for key := range watchedFilesHashes {
			watchedFilesHashes[key] = ""
			_ = watcher.Remove(key)
		}
	})

	// close/end all watches
	Listen("file:watch:close", func(message *Message) {
		endWatch <- true
		<- endWatch
		close(endWatch)
		watchedFilesHashes = nil
		err := watcher.Close()
		if err != nil {
			log.Errorf("Could not close watcher: %s", err)
		} else {
			log.Debugf("File change watching service ended")
		}
	})
}

func StopFileWatch() {
	Send("file:watch:close", &Message{})
}