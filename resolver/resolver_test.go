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
		{"google", "google.com."},
		{"google", "google.com."},
		{"google", "cloudflare.com."},
		{"google", "cloudflare.com."},
		{"google", "cloudflare.com."},
		{"google", "cloudflare.com."},
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

		// get cache
		cache := resolvers.Cache()

		// make sure a response is found
		_, found := cache.Query(d.resolverName, m)
		if !found {
			t.Errorf("Could not find an answer in resolver \"%s\" for %s in the cache", d.resolverName, m)
			continue
		}
	}

}

func TestHostnameResolver(t *testing.T) {
	// load configuration
	conf := testutil.Conf(t, "testdata/hostresolvers.yml")
	resolvers := NewResolverMap(conf.Resolvers)

	data := []struct {
		domain    string
		expectedA string
	}{
		{"router.", "10.0.0.1"},
		{"thing.", "10.0.1.2"},
		{"db.", "10.1.1.3"},
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
		response, err := resolvers.Answer("default", m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		// check response
		if len(response.Answer) < 1 {
			t.Errorf("No answers for question:\n%s\n-----\n%s", m, response)
			continue
		}

		// make sure answer matches as expected
		answer := response.Answer[0].(*dns.A)
		if answer.A.String() != d.expectedA {
			t.Errorf("Expected address does not match answer address: %s != %s", d.expectedA, answer.A.String())
			continue
		}
		if answer.Header().Name != d.domain {
			t.Errorf("Expected name does not match answer name: %s != %s", d.domain, answer.Header().Name)
			continue
		}
	}
}
