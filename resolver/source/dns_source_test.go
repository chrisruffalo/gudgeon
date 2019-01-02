package source

import (
	"testing"

	"github.com/miekg/dns"
)

func TestDnsSourceResolution(t *testing.T) {
	data := []struct {
		domain        string
		serverAddress string
	}{
		// udp, regular port
		{"google.com.", "8.8.8.8"},
		{"cloudflare.com.", "1.1.1.1"},
		// udp, alternate ports
		{"google.com.", "208.67.222.222:5353"},
	}

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
		m.Question[0] = dns.Question{Name: d.domain, Qtype: dns.TypeA, Qclass: dns.ClassINET}

		// create source
		source := newDnsSource(d.serverAddress)

		// use source to resolve
		response, err := source.Answer(m)
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
