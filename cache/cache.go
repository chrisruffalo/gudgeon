package cache

import (
	"time"

	"github.com/miekg/dns"
	backer "github.com/patrickmn/go-cache"
)

// key delimeter
const (
	delimeter = "|"
	dnsMaxTtl = uint32(604800)
)

type envelope struct {
	message *dns.Msg
	time time.Time
}

type Cache interface {
	Store(group string, request *dns.Msg, response *dns.Msg)
	Query(group string, request *dns.Msg) (*dns.Msg, bool)
}

type gocache struct {
	backer *backer.Cache
}

func min(a uint32, b uint32) uint32 {
	if a <= b {
		return a
	}
	return b
}

// make string key from group + message
func key(group string, questions []dns.Question) string {
	key := ""
	if len(questions) > 0 {
		key += group
		for _, question := range questions {
			if len(key) > 0 {
				key += delimeter
			}
			key += question.String()
		}
	}
	return key
}

func New() Cache {
	gocache := new(gocache)
	gocache.backer = backer.New(backer.NoExpiration, 5*time.Minute)

	return gocache
}

func (gocache *gocache) Store(group string, request *dns.Msg, response *dns.Msg) {
	// create key from message
	key := key(group, request.Question)
	if "" == key {
		return
	}

	// get ttl from parts and use lowest ttl as cache value
	ttl := dnsMaxTtl
	for _, value := range response.Answer {
		ttl = min(ttl, value.Header().Ttl)
	}
	for _, value := range response.Ns {
		ttl = min(ttl, value.Header().Ttl)
	}	
	for _, value := range response.Extra {
		ttl = min(ttl, value.Header().Ttl)
	}	

	// if ttl is 0 or less then we don't need to bother to store it at all
	if ttl > 0 {
		// copy response to envelope
		envelope := new(envelope)
		envelope.message = response.Copy()
		envelope.time = time.Now()

		// put in backing store key -> envelope
		gocache.backer.Set(key, envelope, time.Duration(ttl) * time.Second)
	}
}

func (gocache *gocache) Query(group string, request *dns.Msg) (*dns.Msg, bool) {
	// get key
	key := key(group, request.Question)
	if "" == key {
		return nil, false
	}

	value, found := gocache.backer.Get(key)
	if !found {
		return nil, false
	}
	envelope := value.(*envelope)
	if envelope == nil || envelope.message == nil {
		return nil, false
	}
	delta := time.Now().Sub(envelope.time)
	message := envelope.message.Copy()

	// count down/change ttl values in response
	for _, value := range envelope.message.Answer {
		value.Header().Ttl = value.Header().Ttl - uint32(delta/time.Second)
	}
	for _, value := range envelope.message.Ns {
		value.Header().Ttl = value.Header().Ttl - uint32(delta/time.Second)
	}	
	for _, value := range envelope.message.Extra {
		value.Header().Ttl = value.Header().Ttl - uint32(delta/time.Second)
	}	

	return message, true
}