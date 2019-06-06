package resolver

import (
	"fmt"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

type multiSource struct {
	name       string
	sources    []Source
	idx        int

	askChan    chan bool
	chosenChan chan Source
}

func newMultiSource(name string, sources []Source) Source {
	ms := &multiSource{
		name: name,
		sources: sources,
	}
	return ms
}

func (ms *multiSource) Load(specification string) {
	// no-op
}

func (ms *multiSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	for _, source := range ms.sources {
		response, err := source.Answer(rCon, context, request)
		if err == nil && !util.IsEmptyResponse(response) {
			if context != nil {
				context.SourceUsed = ms.Name() + "(" + source.Name() + ")"
			}
			return response, nil
		}
	}
	return nil, fmt.Errorf("No source in multisource: '%s' had a response", ms.name)
}

func (ms *multiSource) Name() string {
	return "ms:" + ms.name
}