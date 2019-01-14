package qlog

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/resolver"
)

type logMsg struct {
	address  *net.IP
	request  *dns.Msg
	response *dns.Msg
	blocked  bool
	result   *resolver.ResolutionResult
	rCon     *resolver.RequestContext
}

func logger(c chan *logMsg) {
	for c := range logChan {
		// get values
		address := c.address
		request := c.request
		response := c.response
		blocked := c.blocked
		result := c.result
		rCon := c.rCon

		// log result if found
		logPrefix := fmt.Sprintf("[%s/%s] q:|%s|%s|->", address.String(), rCon.Protocol, request.Question[0].Name, dns.Type(request.Question[0].Qtype).String())
		if result != nil {
			logSuffix := "->"
			if len(response.Answer) > 0 {
				responseString := strings.TrimSpace(response.Answer[0].String())
				responseLen := len(responseString)
				headerString := strings.TrimSpace(response.Answer[0].Header().String())
				headerLen := len(headerString)
				if responseLen > 0 && headerLen < responseLen {
					logSuffix += responseString[headerLen:]
					if len(response.Answer) > 1 {
						logSuffix += fmt.Sprintf(" (+%d)", len(response.Answer)-1)
					}
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
		} else if blocked {
			fmt.Printf("%s BLOCKED\n", logPrefix)
		} else if response.Rcode == dns.RcodeServerFailure {
			fmt.Printf("%s SERVFAIL:[%s]\n", logPrefix, result.Message)
		} else {
			fmt.Printf("%s RESPONSE[%s]\n", logPrefix, dns.RcodeToString[response.Rcode])
		}
	}
}

// create emtpy chan
var logChan chan *logMsg = nil

func Log(address net.IP, request *dns.Msg, response *dns.Msg, blocked bool, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	if logChan == nil {
		logChan = make(chan *logMsg)
		go logger(logChan)
	}

	// create message and send
	msg := new(logMsg)
	msg.address = &address
	msg.request = request
	msg.response = response
	msg.blocked = blocked
	msg.result = result
	msg.rCon = rCon
	logChan <- msg
}
