package qlog

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/miekg/dns"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/chrisruffalo/gudgeon/resolver"
)

const (
	metricsPrefix  = "gudgeon"
	loggerRoutines = 2
	metricRoutines = 2
)

type logMsg struct {
	address  *net.IP
	request  *dns.Msg
	response *dns.Msg
	result   *resolver.ResolutionResult
	rCon     *resolver.RequestContext
}

// init counters in default registry
func metric(input chan *logMsg) {
	for c := range input {
		queryMeter := metrics.GetOrRegisterMeter(metricsPrefix+"-total-queries", metrics.DefaultRegistry)
		queryMeter.Mark(1)
		if c.result != nil && c.result.Cached {
			cachedMeter := metrics.GetOrRegisterMeter(metricsPrefix+"-total-cache-hits", metrics.DefaultRegistry)
			cachedMeter.Mark(1)
		}
		if c.result != nil && c.result.Blocked {
			blockedMeter := metrics.GetOrRegisterMeter(metricsPrefix+"-blocked-queries", metrics.DefaultRegistry)
			blockedMeter.Mark(1)
		}
	}
}

func logger(input chan *logMsg) {
	for c := range input {
		// get values
		address := c.address
		request := c.request
		response := c.response
		result := c.result
		rCon := c.rCon

		// log result if found
		logPrefix := fmt.Sprintf("[%s/%s] q:|%s|%s|->", address.String(), rCon.Protocol, request.Question[0].Name, dns.Type(request.Question[0].Qtype).String())
		if result != nil {
			logSuffix := "->"
			if result.Blocked {
				listName := "UNKNOWN"
				if result.BlockedList != nil {
					listName = result.BlockedList.CanonicalName()
				}
				ruleText := result.BlockedRule
				fmt.Printf("%s BLOCKED[%s|%s]\n", logPrefix, listName, ruleText)
			} else {
				if len(response.Answer) > 0 {
					responseString := strings.TrimSpace(response.Answer[0].String())
					responseLen := len(responseString)
					headerString := strings.TrimSpace(response.Answer[0].Header().String())
					headerLen := len(headerString)
					if responseLen > 0 && headerLen < responseLen {
						logSuffix += strings.TrimSpace(responseString[headerLen:])
						if len(response.Answer) > 1 {
							logSuffix += fmt.Sprintf(" (+%d)", len(response.Answer)-1)
						}
					} else {
						logSuffix += "(EMPTY RESPONSE)"
					}
				}

				// nothing appended so look at SOA
				if strings.TrimSpace(logSuffix) == "->" {
					if len(response.Ns) > 0 && response.Ns[0].Header().Rrtype == dns.TypeSOA && len(response.Ns[0].String()) > 0 {
						logSuffix += response.Ns[0].(*dns.SOA).Ns
						if len(response.Ns) > 1 {
							logSuffix += fmt.Sprintf(" (+%d)", len(response.Ns)-1)
						}
					} else {
						logSuffix += "(EMPTY)"
					}
				}

				if result.Cached {
					fmt.Printf("%sc:[%s]%s\n", logPrefix, result.Resolver, logSuffix)
				} else {
					fmt.Printf("%sr:[%s]->s:[%s]%s\n", logPrefix, result.Resolver, result.Source, logSuffix)
				}
			}
		} else if response.Rcode == dns.RcodeServerFailure {
			fmt.Printf("%s SERVFAIL:[%s]\n", logPrefix, result.Message)
		} else {
			fmt.Printf("%s RESPONSE[%s]\n", logPrefix, dns.RcodeToString[response.Rcode])
		}
	}
}

// create empty chan
var logChan chan *logMsg
var metChan chan *logMsg

var mux sync.Mutex

func Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	// an optimization step (no need to lock in the later event that this is created)
	if logChan == nil || metChan == nil {
		mux.Lock()
		// only continue if logChan is still nil
		if logChan == nil {
			logChan = make(chan *logMsg)
			// add logger routines
			for rts := 0; rts < loggerRoutines; rts++ {
				go logger(logChan)
			}
		}
		if metChan == nil {
			metChan = make(chan *logMsg)
			// add logger routines
			for rts := 0; rts < metricRoutines; rts++ {
				go metric(metChan)
			}
		}
		mux.Unlock()
	}

	// create message for sending to various endpoints
	msg := new(logMsg)
	msg.address = address
	msg.request = request
	msg.response = response
	msg.result = result
	msg.rCon = rCon

	// send message to metrics and logging channels
	logChan <- msg
	metChan <- msg
}
