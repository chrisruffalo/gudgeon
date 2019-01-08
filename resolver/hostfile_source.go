package resolver

import (
	"net"
	"strings"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	ttl        = 0 // default to never cache since this is basically a free action
	wildcard   = "*"
	comment    = "#"
	altComment = "//"
)

type hostFileSource struct {
	filePath      string
	hostEntries   map[string][]*net.IP
	reverseLookup map[string][]string
	dnsWildcards  map[string][]*net.IP
}

func newHostFileSource(sourceFile string) Source {
	source := new(hostFileSource)
	source.filePath = sourceFile

	// make new map
	source.hostEntries = make(map[string][]*net.IP)
	source.reverseLookup = make(map[string][]string)
	source.dnsWildcards = make(map[string][]*net.IP)

	// open file and parse each line
	data, err := util.GetFileAsArray(sourceFile)

	// on error return empty source
	if err != nil {
		// todo: logging
		return source
	}

	// parse each line
	for _, d := range data {
		// trim whitespace
		d = strings.TrimSpace(d)

		// skip empty strings or strings that start with a comment
		if "" == d || strings.HasPrefix(d, wildcard) || strings.HasPrefix(d, comment) || strings.HasPrefix(d, altComment) {
			continue
		}

		// condition string, all whitespace replaced with actual literal " "
		d = strings.Replace(d, "\t", " ", -1)

		// commas too
		d = strings.Replace(d, ",", " ", -1)

		// remove multiple adjacent spaces
		newstring := ""
		for newstring != d {
			newstring = d
			d = strings.Replace(d, "  ", " ", -1)
		}

		// split after first space
		values := strings.SplitN(d, " ", 2)

		// need at least two values to continue
		if len(values) < 2 {
			continue
		}

		// get domain
		address := values[0]
		address = strings.TrimSpace(address)
		parsedAddress := net.ParseIP(address)
		if parsedAddress == nil {
			// todo: log skipping address
			continue
		}

		// parse out list of domains
		domains := strings.Split(values[1], " ")

		// add to reverse lookup
		ptr := util.ReverseLookupDomain(&parsedAddress)
		source.reverseLookup[ptr] = domains

		// add to map
		for _, domain := range domains {
			if !strings.HasSuffix(domain, ".") {
				domain = domain + "."
			}

			// append value to list
			source.hostEntries[domain] = append(source.hostEntries[domain], &parsedAddress)
		}
	}

	return source
}

func (hostFileSource *hostFileSource) respondToA(name string, response *dns.Msg) {
	// if the domain is available from the host file, go through it
	if val, ok := hostFileSource.hostEntries[name]; ok {
		response.Answer = make([]dns.RR, len(val))

		// entries were found so we need to loop through them
		for idx, address := range val {
			// skip nil addresses
			if address == nil {
				continue
			}

			// create response based on parsed address type (ipv6 or not)
			ipV4 := address.To4()
			ipV6 := address.To16()

			if ipV4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
					A:   ipV4,
				}
				response.Answer[idx] = rr
			} else if ipV6 != nil {
				rr := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: ipV6,
				}
				response.Answer[idx] = rr
			}

		}
	}
}

func (hostFileSource *hostFileSource) respondToPTR(name string, response *dns.Msg) {
	// if the domain is available from the host file, go through it
	if val, ok := hostFileSource.reverseLookup[name]; ok {
		response.Answer = make([]dns.RR, len(val))

		// entries were found so we need to loop through them
		for idx, ptr := range val {
			// skip empty ptr
			if "" != ptr {
				continue
			}

			rr := &dns.PTR{
				Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
				Ptr:   ptr,
			}
			response.Answer[idx] = rr
		}
	}
}

func (hostFileSource *hostFileSource) Answer(context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// return nil response if no question was formed
	if len(request.Question) < 1 {
		return nil, nil
	}

	// get details from question
	question := request.Question[0]
	name := question.Name
	qType := question.Qtype

	// can only respond to A, AAAA, and PTR questions
	if qType != dns.TypeA && qType != dns.TypeAAAA && qType != dns.TypePTR {
		return nil, nil
	}

	// create new response message
	response := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     true,
			AuthenticatedData: true,
			CheckingDisabled:  true,
			RecursionDesired:  true,
			Opcode:            dns.OpcodeQuery,
		},
	}
	response.SetReply(request)

	// handle appropriate question type
	if qType == dns.TypeA || qType == dns.TypeAAAA {
		hostFileSource.respondToA(name, response)
	} else if qType == dns.TypePTR {
		hostFileSource.respondToPTR(name, response)
	}

	return response, nil
}
