package resolver

import (
	"github.com/chrisruffalo/gudgeon/config"
	"net"
	"os"
	"strings"

	"github.com/ryanuber/go-glob"
	log "github.com/sirupsen/logrus"

	"github.com/miekg/dns"
)

const (
	ttl = 60 // default to a small ttl because some things (fire tv/kodi I'm looking at you) will hammer the DNS
)

type Source interface {
	Name() string
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
	// a source that exists as a file is a hostfile source
	if _, err := os.Stat(sourceSpecification); !os.IsNotExist(err) {
		if glob.Glob("*.db", sourceSpecification) {
			source, err := newZoneSourceFromFile(sourceSpecification)
			if err != nil {
				log.Errorf("Loading zone file: %s", err)
			}
			return source
		}

		return newHostFileSource(sourceSpecification)
	}

	// a source that is an IP or that has other hallmarks of an address is a dns source
	if ip := net.ParseIP(sourceSpecification); ip != nil || strings.Contains(sourceSpecification, ":") || strings.Contains(sourceSpecification, "/") {
		return newDNSSource(sourceSpecification)
	}

	// fall back to looking for a resolver source which will basically be a no-op resolver in the event
	// that the named resolver doesn't exist
	return newResolverSource(sourceSpecification)
}
