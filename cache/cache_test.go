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
	msg := new(dns.Msg)

	question := dns.Question{"google.com.", dns.TypeA, dns.ClassINET}
	msg.Question = append(msg.Question, question)

	// create an answer
	answer := &dns.A{
		Hdr: dns.RR_Header{Name: "google.com", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 0},
		A:   net.ParseIP("192.168.0.1"),
	}
	msg.Answer = append(msg.Answer, answer)
	

	cache.Store("default", msg)

	// strip answer section
	msg.Answer = make([]dns.RR, 0)

	found := cache.Query("default", msg)
	if !found {
		t.Errorf("Could not find expected question answer")
	}
}