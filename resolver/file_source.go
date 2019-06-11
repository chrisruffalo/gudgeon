package resolver

import (
	"sync"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/events"
)

type fileSource struct {
	// path to underlying file
	path string
	// is it being watched
	watched bool
	// source that is reloaded
	reloadableSource Source
	// mutex to block source during reload
	mux sync.RWMutex
}



// pretty much directly from https://github.com/fsnotify/fsnotify/blob/master/example_test.go
func (source *fileSource) watchAndLoad() {
	events.Listen("file:" + source.path, func(message *events.Message) {
		log.Infof("Loading new source from: '%s'", source.path)
		source.Load(source.path)
		// notify of source change
		events.Send("souce:change", &events.Message{ "source": source.Name()})
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
		// if not watched start the watch
		if !source.watched {
			source.watchAndLoad()
			source.watched = true
		}
		source.mux.Unlock()
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
