package resolver

import (
	"fmt"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

type lbSource struct {
	name    string
	sources []Source
	idx     int

	askChan    chan bool
	chosenChan chan Source
	closeChan  chan bool
}

func newLoadBalancingSource(name string, sources []Source) Source {
	lb := &lbSource{
		name:       name,
		sources:    sources,
		idx:        0,
		askChan:    make(chan bool),
		chosenChan: make(chan Source),
		closeChan:  make(chan bool),
	}
	go lb.router()
	return lb
}

func (lb *lbSource) Load(specification string) {
	// deliberate no-op
}

func (lb *lbSource) router() {
	for {
		select {
		case <-lb.askChan:
			lb.chosenChan <- lb.sources[lb.idx]
			lb.idx = (lb.idx + 1) % len(lb.sources)
		case <-lb.closeChan:
			lb.closeChan <- true
			return
		}
	}
}

func (lb *lbSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	tries := len(lb.sources)
	for tries >= 0 {
		lb.askChan <- true
		source := <-lb.chosenChan

		response, err := source.Answer(rCon, context, request)
		if err == nil && !util.IsEmptyResponse(response) {
			if context != nil {
				context.SourceUsed = lb.Name() + "(" + source.Name() + ")"
			}
			return response, nil
		}

		// reduce number of tries
		tries--
	}

	return nil, fmt.Errorf("Could not answer question in %d tries", len(lb.sources))
}

func (lb *lbSource) Name() string {
	return "lb:" + lb.name
}

func (lb *lbSource) Close() {
	lb.closeChan <- true
	<-lb.closeChan
	close(lb.askChan)
	close(lb.chosenChan)

	for idx := 0; idx < len(lb.sources); idx++ {
		lb.sources[idx].Close()
	}
}
