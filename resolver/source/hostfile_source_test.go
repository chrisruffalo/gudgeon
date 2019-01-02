package source

import (
	"testing"

	"github.com/miekg/dns"
)

func TestBasicHostFile(t *testing.T) {

	// load sources
	source := newHostFileSource("testdata/test.hosts")

	data := []struct {
		domain string
		expectedAnswers int
	}{
		{"google.com.", 4},
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

		// use source to resolve
		response, err := source.Answer(m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		// check response
		if len(response.Answer) != d.expectedAnswers {
			t.Errorf("Expected %d answers for question but got %d:\n%s\n-----\n%s", d.expectedAnswers, len(response.Answer), m, response)
			continue
		}


	}

}
