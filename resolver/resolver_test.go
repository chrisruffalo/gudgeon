package resolver

import (
	"testing"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestDnsResolver(t *testing.T) {
	// load configuration
	conf := testutil.Conf(t, "testdata/resolvers.yml")
	resolvers := NewResolverMap(conf.Resolvers)

	data := []struct {
		resolverName string
		domain       string
	}{
		{"google", "google.com."},
		{"google", "cloudflare.com."},
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
		response, err := resolvers.Answer(d.resolverName, m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		if response == nil {
			t.Errorf("Got no/nil response from resolver: %s", d.resolverName)
			continue
		}

		// check response
		if len(response.Answer) < 1 {
			t.Errorf("No answers for question:\n%s\n-----\n%s", m, response)
			continue
		}
	}

}
