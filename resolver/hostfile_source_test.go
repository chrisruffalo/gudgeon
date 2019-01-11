package resolver

import (
	"testing"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

func TestBasicHostFile(t *testing.T) {

	// load sources
	source := newHostFileSource("testdata/test.hosts")

	data := []struct {
		domain          string
		qType           uint16
		expectedAnswers int
	}{
		{"google.com.", dns.TypeA, 4},
		{"google2.com.", dns.TypeA, 3},
		{"docs.google.com.", dns.TypeA, 1},
		{"unity.google.com.", dns.TypeA, 1},
		{util.ReverseLookupDomainString("74.125.21.101"), dns.TypePTR, 2},
		{util.ReverseLookupDomainString("2607:f8b0:4002:c09::8a"), dns.TypePTR, 3},
		{"bing.com.", dns.TypeCNAME, 1},
		{"www.bing.com.", dns.TypeCNAME, 1},
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
		m.Question[0] = dns.Question{Name: d.domain, Qtype: d.qType, Qclass: dns.ClassINET}

		// use source to resolve
		response, err := source.Answer(nil, m)
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
