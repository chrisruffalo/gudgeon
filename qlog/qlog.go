package qlog

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/resolver"
)

func Log(address net.IP, request *dns.Msg, response *dns.Msg, blocked bool, result *resolver.ResolutionResult) {
	// log result if found
	logPrefix := fmt.Sprintf("[%s] q:|%s|%s|->", address.String(), request.Question[0].Name, dns.Type(request.Question[0].Qtype).String())
	if result != nil {
		logSuffix := "->"
		if len(response.Answer) > 0 {
			logSuffix += response.Answer[0].String()[len(response.Answer[0].Header().String()):]
			if len(response.Answer) > 1 {
				logSuffix += fmt.Sprintf(" (+%d)", len(response.Answer)-1)
			}
		}

		// nothing appended so look at SOA
		if strings.TrimSpace(logSuffix) == "->" {
			if len(response.Ns) > 0 && response.Ns[0].Header().Rrtype == dns.TypeSOA {
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
