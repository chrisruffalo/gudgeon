package provider

import (
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/testutil"
	"github.com/chrisruffalo/gudgeon/util"
)

func TestProviderStartStop(t *testing.T) {
	config := testutil.Conf(t, "./testdata/provider-test.yml")

	// prepare engine with config options
	engine, err := engine.NewEngine(config)
	if err != nil {
		t.Errorf("Could not build engine: %s", err)
		return
	}

	// create a new provider and start hosting
	provider := NewProvider(engine)
	err = provider.Host(config, engine)
	if err != nil {
		t.Errorf("Creating test provider: %s", err)
		return
	}
	time.Sleep(5 * time.Second)

	// make sure they shut down
	err = provider.Shutdown()
	if err != nil {
		t.Errorf("Shutting down test provider: %s", err)
	}
}

func TestProviderLocalResolution(t *testing.T) {
	// query data
	data := []struct {
		question string
		expected string
	}{
		{"google.com", "127.0.0.1"},
		{"google.com", "127.0.0.1"},
		{"alias.google.com", "127.0.0.1"},
		{"videos.google.com", "127.0.0.1"},
		{"change.google.com", "127.0.0.1"},
		{"youtube.com", "10.0.0.1"},
		{"alias.youtube.com", "10.0.0.1"},
	}

	// create from config
	config := testutil.Conf(t, "./testdata/provider-test.yml")

	// prepare engine with config options
	engine, err := engine.NewEngine(config)
	if err != nil {
		t.Errorf("Could not build engine: %s", err)
		return
	}

	// create a new provider and start hosting
	provider := NewProvider(engine)
	err = provider.Host(config, engine)
	if err != nil {
		t.Errorf("Creating test provider: %s", err)
		return
	}
	time.Sleep(5 * time.Second)

	// create dns sources and use it
	sources := []resolver.Source{resolver.NewSource("127.0.0.1:25353/tcp"), resolver.NewSource("127.0.0.1:25353/tcp")}

	// use each source on each data element
	for _, d := range data {
		// create dns message
		m := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative:     true,
				AuthenticatedData: true,
				RecursionDesired:  true,
				Opcode:            dns.OpcodeQuery,
			},
		}

		// make question parts
		m.Question = make([]dns.Question, 1)
		m.Question[0] = dns.Question{Name: dns.Fqdn(d.question), Qtype: dns.TypeA, Qclass: dns.ClassINET}

		for _, source := range sources {
			// query and check using each source
			rCon := resolver.DefaultRequestContext()
			response, err := source.Answer(rCon, nil, m)
			if err != nil {
				t.Errorf("Could not resolve question '%s' using source %s: %s", d.question, source.Name(), err)
				continue
			}

			// check response
			if response == nil {
				t.Errorf("Nil response for question '%s' from source %s:\n%s\n-----\n%s", d.question, source.Name(), m, response)
				continue
			}

			if len(response.Answer) < 1 {
				t.Errorf("No aswers for question '%s' from source %s:\n%s\n-----\n%s", d.question, source.Name(), m, response)
				continue
			}

			// check first record
			first := util.GetFirstIPResponse(response)
			if d.expected != first {
				t.Errorf("Expected answer '%s' but got '%s' for question '%s'", d.expected, first, d.question)
			}
		}
	}

	// make sure they shut down
	err = provider.Shutdown()
	if err != nil {
		t.Errorf("Shutting down test provider: %s", err)
	}

	// close sources
	for _, source := range sources {
		source.Close()
	}
}
