package cache

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

func TestSimpleCache(t *testing.T) {
	// create new cache
	cache := New()

	// create a new msg
	request := new(dns.Msg)

	question := dns.Question{"google.com.", dns.TypeA, dns.ClassINET}
	request.Question = append(request.Question, question)

	// create an answer
	response := request.Copy()
	answer := &dns.A{
		Hdr: dns.RR_Header{Name: "google.com", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300},
		A:   net.ParseIP("192.168.0.1"),
	}
	response.Answer = append(response.Answer, answer)

	cache.Store("default", request, response)

	_, found := cache.Query("default", request)
	if !found {
		t.Errorf("Could not find expected question answer")
	}
}
