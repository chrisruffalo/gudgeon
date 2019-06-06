package resolver

import (
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"

	"github.com/chrisruffalo/gudgeon/config"
)

const (
	ttl = 60 // default to a small ttl because some things (fire tv/kodi I'm looking at you) will hammer the DNS
)

type Source interface {
	Name() string
	Load(specification string)
	Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
}

func NewConfigurationSource(config *config.GudgeonSource, sourceMap map[string]Source) Source {

	// create an array and guess at final size
	sources := make([]Source, 0, len(config.Specs))

	// for each spec create a source if it isn't in the source map
	for _, spec := range config.Specs {
		var newSource Source
		if sourceMap != nil {
			item, found := sourceMap[spec]
			if found {
				newSource = item
			}
		}
		// source not found in map
		if newSource == nil {
			newSource = NewSource(spec)
		}
		if newSource != nil {
			// add source to list of sources that will be used by balancer or list
			sources = append(sources, newSource)

			// update source in map, essentially a no-op in most cases
			if sourceMap != nil {
				sourceMap[spec] = newSource
			}
		}
	}
	// if multiple sources are defined
	if len(sources) > 1 {
		if config.LoadBalance {
			return newLoadBalancingSource(config.Name, sources)
		} else {
			return newMultiSource(config.Name, sources)
		}
	} else if len(sources) > 0 {
		return sources[0]
	}
	// if no source specs are provided, return nil
	return nil
}

func NewSource(sourceSpecification string) Source {
	var source Source

	// a source that exists as a file is a either a db(zone file), resolv.conf, or hostfile source
	if _, err := os.Stat(sourceSpecification); !os.IsNotExist(err) {
		// put reloadable/file watching source in the middle
		watcher := &fileSource{}
		// determine type of file source
		if glob.Glob("*.db", sourceSpecification) {
			watcher.reloadableSource = &zoneSource{}
		} else {
			// the fallback is to treat it as a hostfile
			watcher.reloadableSource = &hostFileSource{}
		}
		source = watcher
	} else if ip := net.ParseIP(sourceSpecification); ip != nil || strings.Contains(sourceSpecification, ":") || strings.Contains(sourceSpecification, "/") {
		// a source that is an IP or that has other hallmarks of an address is a dns source
		source = &dnsSource{}
	}

	// fall back to looking for a resolver source which will basically be a no-op resolver in the event
	// that the named resolver doesn't exist
	if source == nil {
		source = &resolverSource{}
	}

	// load source
	source.Load(sourceSpecification)

	// finally return
	return source
}
