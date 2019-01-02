package source

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
	filePath    string
	hostEntries map[string][]*net.IP
}

func newHostFileSource(sourceFile string) Source {
	source := new(hostFileSource)
	source.filePath = sourceFile

	// make new map
	source.hostEntries = make(map[string][]*net.IP)

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
		values := strings.Split(d, " ")

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

		// parse out list
		domains := strings.Split(values[1], " ")

		// add to map
		for _, domain := range domains {
			if !strings.HasSuffix(domain, ".") {
				domain = domain + "."
			}

			// if no value exists, make new list
			if _, ok := source.hostEntries[domain]; !ok {
				source.hostEntries[domain] = make([]*net.IP, 0)
			}

			// append value to list
			source.hostEntries[domain] = append(source.hostEntries[domain], &parsedAddress)
		}
	}

	return source
}

func (hostFileSource *hostFileSource) Answer(request *dns.Msg) (*dns.Msg, error) {
	// return nil response if no question was formed
	if len(request.Question) < 1 {
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
		Answer: make([]dns.RR, 0),
	}

	// get name from question
	name := request.Question[0].Name

	// if the domain is available from the host file, go through it
	if val, ok := hostFileSource.hostEntries[name]; ok {
		// entries were found so we need to loop through them
		for _, address := range val {
			// skipp nil addresses
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
				response.Answer = append(response.Answer, rr)
			} else if ipV6 != nil {
				rr := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: ipV6,
				}
				response.Answer = append(response.Answer, rr)
			}

		}
	}

	return response, nil
}
