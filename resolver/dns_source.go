package resolver

import (
	"strconv"
	"strings"

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

func (dnsSource *dnsSource) Answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// create new client instance
	client := new(dns.Client)

	// forward message without interference
	response, _, err := client.Exchange(request, dnsSource.remoteAddress)

	// return error if error
	if err != nil {
		return nil, err
	}

	// set reply (pretty sure this is done upstream)
	//response.SetReply(request)

	// otherwise just return response
	return response, nil
}
