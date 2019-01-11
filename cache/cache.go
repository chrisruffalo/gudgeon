package cache

import (
	"time"

	"github.com/miekg/dns"
	backer "github.com/patrickmn/go-cache"
)

// key delimeter
const (
	delimeter                 = "|"
	dnsMaxTtl                 = uint32(604800)
	defaultCacheScrapeMinutes = 1
)

type envelope struct {
	message *dns.Msg
	time    time.Time
}

type Cache interface {
	Store(partition string, request *dns.Msg, response *dns.Msg)
	Query(partition string, request *dns.Msg) (*dns.Msg, bool)
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

func max(a uint32, b uint32) uint32 {
	if a <= b {
		return b
	}
	return a
}

// make string key from partition + message
func key(partition string, questions []dns.Question) string {
	key := ""
	if len(questions) > 0 {
		key += partition
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
	gocache.backer = backer.New(backer.NoExpiration, defaultCacheScrapeMinutes*time.Minute)

	return gocache
}

func minTtl(currentMin uint32, records []dns.RR) uint32 {
	for _, value := range records {
		currentMin = min(currentMin, value.Header().Ttl)
	}
	return currentMin
}

func (gocache *gocache) Store(partition string, request *dns.Msg, response *dns.Msg) {
	// you can't cache a nil response
	if response == nil {
		return
	}

	// don't store an empty response
	if len(response.Answer) < 1 && len(response.Ns) < 1 && len(response.Extra) < 1 {
		return
	}

	// create key from message
	key := key(partition, request.Question)
	if "" == key {
		return
	}

	// get ttl from parts and use lowest ttl as cache value
	ttl := minTtl(dnsMaxTtl, response.Answer)
	if len(response.Answer) < 1 {
		ttl = minTtl(dnsMaxTtl, response.Ns)
		if len(response.Ns) < 1 {
			ttl = minTtl(dnsMaxTtl, response.Extra)
		}
	}

	// if ttl is 0 or less then we don't need to bother to store it at all
	if ttl > 0 {
		// copy response to envelope
		envelope := new(envelope)
		envelope.message = response.Copy()
		envelope.time = time.Now()

		// put in backing store key -> envelope
		gocache.backer.Set(key, envelope, time.Duration(ttl)*time.Second)
	}
}

func adjustTtls(timeDelta uint32, records []dns.RR) {
	for _, value := range records {
		if value.Header().Ttl > timeDelta {
			value.Header().Ttl = value.Header().Ttl - timeDelta
		} else {
			value.Header().Ttl = 0
		}
	}
}

func (gocache *gocache) Query(partition string, request *dns.Msg) (*dns.Msg, bool) {
	// get key
	key := key(partition, request.Question)
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

	// update message id to match request id
	message.MsgHdr.Id = request.MsgHdr.Id

	// count down/change ttl values in response
	secondDelta := uint32(delta / time.Second)
	adjustTtls(secondDelta, envelope.message.Answer)
	adjustTtls(secondDelta, envelope.message.Ns)
	adjustTtls(secondDelta, envelope.message.Extra)

	return message, true
}
