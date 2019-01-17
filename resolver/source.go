package resolver

import (
	"net"
	"os"
	"strings"

	"github.com/miekg/dns"
)

type Source interface {
	Name() string
	Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error)
}

func NewSource(sourceSpecification string) Source {
	// a source that exists as a file is a hostfile source
	if _, err := os.Stat(sourceSpecification); !os.IsNotExist(err) {
		return newHostFileSource(sourceSpecification)
	}

	// a source that is an IP or that has other hallmarks of an address is a dns source
	if ip := net.ParseIP(sourceSpecification); ip != nil || strings.Contains(sourceSpecification, ":") || strings.Contains(sourceSpecification, "/") {
		return newDnsSource(sourceSpecification)
	}

	// fall back to looking for a resolver source which will basically be a no-op resolver in the event
	// that the named resolver doesn't exist
	return newResolverSource(sourceSpecification)
}
