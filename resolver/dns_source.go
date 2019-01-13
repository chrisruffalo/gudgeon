package resolver

import (
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	defaultPort   = uint(53)
	portDelimeter = ":"
)

type dnsSource struct {
	dnsServer     string
	port          uint
	remoteAddress string
}

func newDnsSource(sourceAddress string) Source {
	source := new(dnsSource)
	source.port = 0
	source.dnsServer = sourceAddress

	// need to determine if a port comes along with the address and parse it out once
	if strings.Contains(sourceAddress, portDelimeter) {
		split := strings.Split(sourceAddress, portDelimeter)
		if len(split) > 1 {
			source.dnsServer = split[0]
			var err error
			parsePort, err := strconv.ParseUint(split[1], 10, 32)
			// recover from error
			if err != nil {
				source.port = 0
			} else {
				source.port = uint(parsePort)
			}
		}
	}

	// recover from parse errors or use default port in event port wasn't set
	if source.port == 0 {
		source.port = defaultPort
	}

	// save/parse remote address once
	source.remoteAddress = source.dnsServer + portDelimeter + strconv.Itoa(int(source.port))

	return source
}

func (dnsSource *dnsSource) Name() string {
	return dnsSource.remoteAddress
}

func (dnsSource *dnsSource) query(coType string, request *dns.Msg, remoteAddress string) (*dns.Msg, error) {
	var err error = nil

	co := new(dns.Conn)
	if co.Conn, err = net.DialTimeout(coType, remoteAddress, 2*time.Second); err != nil {
		return nil, err
	}
	defer co.Close()


	// write message
	if err := co.WriteMsg(request); err != nil {
		return nil, err
	}

	return co.ReadMsg()
}

func (dnsSource *dnsSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// forward message without interference
	response, err := dnsSource.query(rCon.Protocol, request, dnsSource.remoteAddress)
	if err != nil {
		// on tcp err fall back to udp
		if rCon.Protocol == "tcp" {
			response, err = dnsSource.query(rCon.Protocol, request, dnsSource.remoteAddress)
		}

		if err != nil {
			return nil, err
		}
	}

	// do not set reply here (doesn't seem to matter, leaving this comment so nobody decides to do it in the future without cause)
	// response.SetReply(request)

	// set source as answering source
	if context != nil {
		context.SourceUsed = dnsSource.Name()
	}

	// otherwise just return
	return response, nil
}
