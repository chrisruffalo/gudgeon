package resolver

import (
	"net"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	wildcard   = "*"
)

type hostFileSource struct {
	filePath      string
	hostEntries   map[string][]*net.IP
	cnameEntries  map[string]string
	reverseLookup map[string][]string
	dnsWildcards  map[string][]*net.IP
}

func newHostFileFromHostArray(data []string) Source {
	source := new(hostFileSource)
	source.filePath = "hosts"

	// make new map
	source.hostEntries = make(map[string][]*net.IP)
	source.cnameEntries = make(map[string]string)
	source.reverseLookup = make(map[string][]string)
	source.dnsWildcards = make(map[string][]*net.IP)

	// parse each line
	for _, d := range data {
		// trim whitespace
		d = strings.TrimSpace(d)
		d = strings.ToLower(d)

		// trim comments
		d = util.TrimComments(d)

		// skip empty strings or strings that start with a wildcard
		if "" == d || strings.HasPrefix(d, wildcard) {
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
		address := strings.TrimSpace(values[0])
		parsedAddress := net.ParseIP(address)

		// parse out list of domains
		domains := strings.Split(values[1], " ")

		if parsedAddress != nil {
			// add to reverse lookup
			ptr := util.ReverseLookupDomain(&parsedAddress)
			source.reverseLookup[ptr] = domains

			// add to map
			for _, domain := range domains {
				// determine if domain is wild or not
				wild := strings.Contains(domain, "*")

				// fully qualify domain
				domain := dns.Fqdn(domain)

				// append value to list
				if !wild {
					source.hostEntries[domain] = append(source.hostEntries[domain], &parsedAddress)
				} else {
					source.dnsWildcards[domain] = append(source.dnsWildcards[domain], &parsedAddress)
				}
			}
		} else {
			// add target to alias cname lookup
			for _, alias := range domains {
				// only one alias per taget
				if "" == source.cnameEntries[alias] {
					source.cnameEntries[dns.Fqdn(alias)] = dns.Fqdn(address)
				}
			}
		}
	}

	return source
}

func newHostFileSource(sourceFile string) Source {
	// open file and parse each line
	data, err := util.GetFileAsArray(sourceFile)
	// on error return nil
	if err != nil {
		log.Errorf("While opening '%s': %s", sourceFile, err)
		return nil
	}

	source := newHostFileFromHostArray(data)
	if source != nil {
		source.(*hostFileSource).filePath = sourceFile
	}

	return source
}

func (hostFileSource *hostFileSource) respondToAWildcards(name string, request *dns.Msg, response *dns.Msg) {
	// only try if there are wildcards
	if len(hostFileSource.dnsWildcards) < 1 {
		return
	}

	// get question type
	questionType := request.Question[0].Qtype

	// now inspect wildcards
	for wildDomain, addresses := range hostFileSource.dnsWildcards {
		// move on if the wildcard doesn't match
		if !glob.Glob(wildDomain, name) {
			continue
		}
		// otherwise create and append records
		for _, address := range addresses {
			// skip nil addresses
			if address == nil {
				continue
			}

			// create response based on parsed address type (ipv6 or not)
			ipV4 := address.To4()
			ipV6 := address.To16()

			if questionType == dns.TypeA && ipV4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
					A:   ipV4,
				}
				response.Answer = append(response.Answer, rr)
			}

			if questionType == dns.TypeAAAA && ipV4 == nil && ipV6 != nil {
				rr := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: ipV6,
				}
				response.Answer = append(response.Answer, rr)
			}
		}
	}
}

func (hostFileSource *hostFileSource) respondToA(name string, request *dns.Msg, response *dns.Msg) {
	// first respond to wildcards
	hostFileSource.respondToAWildcards(name, request, response)

	// get question type
	questionType := request.Question[0].Qtype

	// if the domain is available from the host file, go through it
	if val, ok := hostFileSource.hostEntries[name]; ok {
		// entries were found so we need to loop through them
		for _, address := range val {
			// skip nil addresses
			if address == nil {
				continue
			}

			// create response based on parsed address type (ipv6 or not)
			ipV4 := address.To4()
			ipV6 := address.To16()

			if questionType == dns.TypeA && ipV4 != nil {
				rr := &dns.A{
					Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
					A:   ipV4,
				}
				response.Answer = append(response.Answer, rr)
			}

			if questionType == dns.TypeAAAA && ipV4 == nil && ipV6 != nil {
				rr := &dns.AAAA{
					Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
					AAAA: ipV6,
				}
				response.Answer = append(response.Answer, rr)
			}
		}
	}
}

func (hostFileSource *hostFileSource) respondToPTR(name string, response *dns.Msg) {
	// if the (reverse lookup) domain is available from the host file, go through it
	if val, ok := hostFileSource.reverseLookup[name]; ok {

		// entries were found so we need to loop through them
		for _, ptr := range val {
			// skip empty ptr
			if "" == ptr {
				continue
			}

			ptr = dns.Fqdn(ptr)

			rr := &dns.PTR{
				Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
				Ptr: ptr,
			}
			response.Answer = append(response.Answer, rr)
		}
	}
}

func (hostFileSource *hostFileSource) respondToCNAME(name string, response *dns.Msg) {
	// if the domain is available from the host file, go through it
	if cname, ok := hostFileSource.cnameEntries[name]; ok {
		response.Answer = make([]dns.RR, 1)

		// skip empty ptr
		if "" == cname {
			return
		}

		cname = dns.Fqdn(cname)

		rr := &dns.CNAME{
			Hdr:    dns.RR_Header{Name: name, Rrtype: dns.TypeCNAME, Class: dns.ClassINET, Ttl: ttl},
			Target: cname,
		}
		response.Answer[0] = rr
	}
}

func (hostFileSource *hostFileSource) Name() string {
	return "hostfile:" + hostFileSource.filePath
}

func (hostFileSource *hostFileSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// return nil response if no question was formed
	if len(request.Question) < 1 {
		return nil, nil
	}

	// get details from question
	question := request.Question[0]
	name := strings.ToLower(question.Name)
	qType := question.Qtype

	// can only respond to A, AAAA, PTR, and CNAME questions
	if qType != dns.TypeANY && qType != dns.TypeA && qType != dns.TypeAAAA && qType != dns.TypePTR && qType != dns.TypeCNAME {
		return nil, nil
	}

	// create new response message
	response := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative: true,
			Opcode:        dns.OpcodeQuery,
		},
	}
	response.SetReply(request)

	// handle appropriate question type
	if qType == dns.TypeANY || qType == dns.TypeCNAME {
		hostFileSource.respondToCNAME(name, response)
	}

	if qType == dns.TypeANY || qType == dns.TypeA || qType == dns.TypeAAAA {
		// look for cnames before looking for other names
		if qType != dns.TypeANY {
			hostFileSource.respondToCNAME(name, response)
		}
		// if no cnames are we can look for A/AAAA responses
		if qType == dns.TypeANY || len(response.Answer) < 1 {
			hostFileSource.respondToA(name, request, response)
		}
	}

	if qType == dns.TypeANY || qType == dns.TypePTR {
		hostFileSource.respondToPTR(name, response)
	}

	// make sure case of question matches
	for _, rr := range response.Answer {
		rr.Header().Name = question.Name
	}

	// set source as answering source
	if context != nil && !util.IsEmptyResponse(response) {
		// don't cache responses
		context.Stored = true

		// update source used
		context.SourceUsed = hostFileSource.Name()
	}

	return response, nil
}
