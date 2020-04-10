package resolver

import (
	log "github.com/sirupsen/logrus"
	"testing"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestDnsResolver(t *testing.T) {
	// load configuration
	conf := testutil.TestConf(t, "testdata/resolvers.yml")
	resolvers := NewResolverMap(conf, conf.Resolvers)

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
		response, _, err := resolvers.Answer(nil, d.resolverName, m)
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

	resolvers.Close()
}

func TestHostnameResolver(t *testing.T) {
	// load configuration
	conf := testutil.TestConf(t, "testdata/hostresolvers.yml")
	resolvers := NewResolverMap(conf, conf.Resolvers)

	data := []struct {
		resolver  string
		domain    string
		expectedA string
	}{
		{"default", "router.", "10.0.0.1"},
		{"default", "thing.", "10.0.1.2"},
		{"default", "db.", "10.1.1.3"},
		{"testskip", "noskip.", "10.0.10.1"},
		{"testskip", "skip.lan.", "10.0.10.2"},
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
		response, _, err := resolvers.Answer(nil, d.resolver, m)
		if err != nil {
			t.Errorf("Could not resolve: %s", err)
			continue
		}

		// make sure response is not nil
		if response == nil {
			t.Errorf("Nil response for question:\n%s\n-----\n%s", m, response)
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

	resolvers.Close()
}

func BenchmarkResolver(b *testing.B) {
	// load configuration
	conf := testutil.BenchConf(b, "testdata/resolvers.yml")

	resolvers := NewResolverMap(conf, conf.Resolvers)

	questions := []string{
		"google.com.",
		"reddit.com.",
		"microsoft.com.",
	}

	// start timer
	b.ResetTimer()
	b.ReportAllocs()

	// benchmark query
	b.RunParallel(func(pb *testing.PB) {
		idx := 0
		for pb.Next() {
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
			m.Question[0] = dns.Question{Name: questions[idx%len(questions)], Qtype: dns.TypeA, Qclass: dns.ClassINET}
			idx++

			rCon := DefaultRequestContext()
			// use source to resolve
			_, result, err := resolvers.Answer(rCon, "default", m)
			if err != nil {
				log.Errorf("Could not resolve: %s", err)
			}
			// accurately reflect what is being done in real use
			rCon.Put()
			result.Put()
			//log.Infof("idx: %d", idx)
		}
	})

	resolvers.Close()
}
