package resolver

import (
	"testing"

	"github.com/miekg/dns"
)

// warning: this **requires** a working system resolver
func TestSystemSourceResolution(t *testing.T) {
	data := []struct {
		question string
		qtype    uint16
	}{
		{"google.com.", dns.TypeA},
		{"cloudflare.com.", dns.TypeA},
		{"cloudflare.com.", dns.TypeCNAME},
		{"8.8.8.8", dns.TypePTR},
	}

	// create source
	source := newSystemSource()

	for _, d := range data {
		// create dns message from scratch
		m := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative:     true,
				AuthenticatedData: true,
				CheckingDisabled:  true,
				RecursionDesired:  true,
				Opcode:            dns.OpcodeQuery,
			},
			Question: make([]dns.Question, 1),
		}
		m.Question[0] = dns.Question{Name: d.question, Qtype: d.qtype, Qclass: dns.ClassINET}

		// use source to resolve
		rCon := DefaultRequestContext()
		rCon.Protocol = "udp"
		response, err := source.Answer(rCon, nil, m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		// check response
		if len(response.Answer) < 1 {
			t.Errorf("No answers for question:\n%s\n-----\n%s", m, response)
			continue
		}
	}
}
