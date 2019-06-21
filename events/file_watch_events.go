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

func startFileWatch() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Errorf("While creating watcher: %s", err)
	}
	// todo: find another way to close
	//defer watcher.Close()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if _, err := os.Stat(event.Name); err != nil {
					log.Infof("Error accessing file, eating event: %s", err)
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
					log.Debugf("Watching file: %s", path)
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

	// end all watches
	Listen("file:watch:clear", func(message *Message) {
		for key := range watchedFilesHashes {
			watchedFilesHashes[key] = ""
			_ = watcher.Remove(key)
		}
	})
}

// start the file watch
// todo: fix this, this is really really really bad practice
func init() {
	startFileWatch()
}