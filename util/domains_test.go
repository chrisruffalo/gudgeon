package util

import (
	"net"
	"testing"
)

func TestReverseLookup(t *testing.T) {

	data := []struct {
		input    string
		expected string
	}{
		{"127.0.0.1", "1.0.0.127.in-addr.arpa."},
		{"::1", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa."},
		{"2001:db8::1", "1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.8.b.d.0.1.0.0.2.ip6.arpa."},
	}

	for _, d := range data {
		ip := net.ParseIP(d.input)
		result := ReverseLookupDomain(&ip)
		if result != d.expected {
			t.Errorf("Reverse lookup domain \"%s\" did not match expected domain \"%s\"", result, d.expected)
		}
	}
}

func TestReverseDomain(t *testing.T) {
	data := []struct {
		input    string
		expected string
	}{
		{"tld", "tld"},
		{"tld.", "tld"},
		{"thing.tld", "tld.thing"},
		{"two.one.thing.tld.", "tld.thing.one.two"},
		{"              five.four.three.two.one.thing.tld.", "tld.thing.one.two.three.four.five"},
		{"", ""},
		{"thing", "thing"},
	}

	for _, d := range data {
		output := ReverseDomainTree(d.input)
		if output != d.expected {
			t.Errorf("Reverse domain expected '%s' but got '%s'", d.expected, output)
		}
	}
}
