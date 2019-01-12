package qlog

import (
	"fmt"
	"net"

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

		if result.Cached {
			fmt.Printf("%sc:[%s]%s\n", logPrefix, result.Resolver, logSuffix)
		} else {
			fmt.Printf("%sr:[%s]->s:[%s]%s\n", logPrefix, result.Resolver, result.Source, logSuffix)
		}
	} else if blocked {
		fmt.Printf("%s BLOCKED\n", logPrefix)
	} else {
		fmt.Printf("%s NXDOMAIN\n", logPrefix)
	}
}
