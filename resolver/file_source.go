package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

)

type fileSource struct {
	// path to underlying file
	path string
	// sha2-256 of file, so file isn't reloaded just for being touched
	pathHash string
	// source that is reloaded
	reloadableSource Source
	// mutex to block source during reload
	mux sync.RWMutex
}

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

// pretty much directly from https://github.com/fsnotify/fsnotify/blob/master/example_test.go
func (source *fileSource) watchAndLoad() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case _, ok := <-watcher.Events:
				if !ok {
					return
				}
				//log.Infof("event:", event)
				newHash := pathHash(source.path)
				if "" != newHash && newHash != source.pathHash {
					source.pathHash = newHash
					log.Infof("Loading new source from: '%s'", source.path)
					source.Load(source.path)
					err = watcher.Add(source.path)
					if err != nil {
						log.Errorf("Error watching '%s': %s", source.path, err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Errorf("Error: %s", err)
			}
		}
	}()

	err = watcher.Add(source.path)
	if err != nil {
		log.Errorf("Error watching '%s': %s", source.path, err)
	}
	<-done
}

func (source *fileSource) Name() string {
	if source.reloadableSource != nil {
		return source.reloadableSource.Name()
	}
	return ""
}

func (source *fileSource) Load(specification string) {
	if source.reloadableSource != nil {
		source.mux.Lock()
		// assumed that specification is already a path to a file here
		source.path = specification
		source.reloadableSource.Load(source.path)
		source.mux.Unlock()
		// only do this part once
		if "" == source.pathHash {
			source.pathHash = pathHash(source.path)
			// re-launch watcher
			go source.watchAndLoad()
		}
	}
}

func (source *fileSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	if source.reloadableSource != nil {
		// reloading the file is the write-half of this mutex
		source.mux.RLock()
		defer source.mux.RUnlock()
		return source.reloadableSource.Answer(rCon, context, request)
	}
	return nil, nil
}
