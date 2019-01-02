package source

import (
	"github.com/miekg/dns"
)

type hostFileSource struct {
	filePath string
}

func newHostFileSource(sourceFile string) Source {
	source := new(hostFileSource)
	source.filePath = sourceFile
	return source
}

func (hostFileSource *hostFileSource) Answer(request *dns.Msg) (*dns.Msg, error) {
	return nil, nil
}