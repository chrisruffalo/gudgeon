package resolver

import (
	"net"
	"strings"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

type systemSource struct {
}

func (source *systemSource) Load(specification string) {
	// deliberate no-op
}

func (source *systemSource) Name() string {
	return "system"
}

func (source *systemSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// get details from question
	question := request.Question[0]
	name := strings.ToLower(question.Name)
	qType := question.Qtype

	// can only respond to A, AAAA, PTR, and CNAME questions
	if qType != dns.TypeA && qType != dns.TypeAAAA && qType != dns.TypePTR && qType != dns.TypeCNAME {
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
	if qType == dns.TypeCNAME || qType == dns.TypeA || qType == dns.TypeAAAA {
		// use system resolver through net package
		hosts, err := net.LookupHost(name)
		// use the found hosts to create A or AAAA records
		if err == nil && len(hosts) > 0 {
			for _, host := range hosts {
				// parse host ip
				address := net.ParseIP(host)

				// skip nil addresses
				if address == nil {
					continue
				}

				// create response based on parsed address type (ipv6 or not)
				ipV4 := address.To4()
				ipV6 := address.To16()

				if (qType == dns.TypeA || qType == dns.TypeCNAME) && ipV4 != nil {
					rr := &dns.A{
						Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
						A:   ipV4,
					}
					response.Answer = append(response.Answer, rr)
				}

				if (qType == dns.TypeAAAA || qType == dns.TypeCNAME) && ipV4 == nil && ipV6 != nil {
					rr := &dns.AAAA{
						Hdr:  dns.RR_Header{Name: name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl},
						AAAA: ipV6,
					}
					response.Answer = append(response.Answer, rr)
				}
			}
		}
	}

	if qType == dns.TypePTR {
		names, err := net.LookupAddr(name)
		if err == nil && len(names) > 0 {
			for _, ptr := range names {
				// skip empty ptr
				if "" == ptr {
					continue
				}

				rr := &dns.PTR{
					Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
					Ptr: dns.Fqdn(ptr),
				}
				response.Answer = append(response.Answer, rr)
			}
		}
	}

	// make sure case of question matches
	for _, rr := range response.Answer {
		rr.Header().Name = question.Name
	}

	// set source as answering source if the source is not nil
	if context != nil && !util.IsEmptyResponse(response) {
		// update source used
		context.SourceUsed = source.Name()
	}

	return response, nil
}

func (source *systemSource) Close() {

}
