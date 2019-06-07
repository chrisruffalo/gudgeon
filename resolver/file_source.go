package resolver

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"sync"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/events"

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
	events.Listen("file:" + source.path, func(message *events.Message) {
		newHash := pathHash(source.path)
		if "" != newHash && newHash != source.pathHash {
			source.pathHash = newHash
			log.Infof("Loading new source from: '%s'", source.path)
			source.Load(source.path)
			// notify of source change
			events.Send("souce:change", &events.Message{ "source": source.Name()})
		}
	})
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
			source.watchAndLoad()
		}
		events.Send("file:watch", &events.Message{"path": source.path})
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
