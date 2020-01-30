package cache

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	backer "github.com/patrickmn/go-cache"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	// pattern to use for building each step of the key
	_keyPattern = "%s|%s|%s"
	// absolute max TTL
	dnsMaxTTL = uint32(604800)
	/// default time to scrape expired items
	defaultCacheScrapeMinutes = 1
)

type envelope struct {
	message *dns.Msg
	time    time.Time
}

type Cache interface {
	Store(partition string, request *dns.Msg, response *dns.Msg) bool
	Query(partition string, request *dns.Msg) (*dns.Msg, bool)
	Size() uint32
	Clear()
}

// keeps a pointer to the backer as well as a map of
// group -> int mappings that saves several bytes for
// each key entry. (this optimization may be overkill)
type gocache struct {
	builderPool  sync.Pool
	backers      map[string]*backer.Cache
	partitionMux sync.RWMutex
}

func min(a uint32, b uint32) uint32 {
	if a <= b {
		return a
	}
	return b
}

func New() Cache {
	return &gocache{
		builderPool: sync.Pool{
			New: func() interface{} {
				return &strings.Builder{}
			},
		},
		backers: make(map[string]*backer.Cache),
	}
}

func minTTL(currentMin uint32, records []dns.RR) uint32 {
	for _, value := range records {
		currentMin = min(currentMin, value.Header().Ttl)
	}
	return currentMin
}

// make string key from partition + message
func (gocache *gocache) key(questions []dns.Question) string {

	if len(questions) > 0 {
		// micro optimization when only one question is asked
		if len(questions) == 1 {
			return strings.ToLower(fmt.Sprintf(_keyPattern, questions[0].Name, dns.Class(questions[0].Qclass).String(), dns.Type(questions[0].Qtype).String()))
		}
		// get pooled string builder and prepare to return to pool after reset
		builder := gocache.builderPool.Get().(*strings.Builder)
		defer func() {
			builder.Reset()
			gocache.builderPool.Put(builder)
		}()
		for idx := 0; idx < len(questions); idx++ {
			if idx > 0 {
				builder.WriteString("|")
			}
			builder.WriteString(fmt.Sprintf(_keyPattern, questions[idx].Name, dns.Class(questions[idx].Qclass).String(), dns.Type(questions[idx].Qtype).String()))
		}
		return strings.ToLower(builder.String())
	}
	return ""
}

func (gocache *gocache) Store(partition string, request *dns.Msg, response *dns.Msg) bool {
	// you shouldn't cache an empty response (or a truncated response)
	if util.IsEmptyResponse(response) || response.MsgHdr.Truncated {
		return false
	}

	// get ttl from parts and use lowest ttl as cache value
	ttl := minTTL(dnsMaxTTL, response.Answer)
	if len(response.Answer) < 1 {
		ttl = minTTL(dnsMaxTTL, response.Ns)
		if len(response.Ns) < 1 {
			ttl = minTTL(dnsMaxTTL, response.Extra)
		}
	}

	// if ttl is 0 or less then we don't need to bother to store it at all
	if ttl > 0 {
		// create key from message
		key := gocache.key(request.Question)
		if "" == key {
			return false
		}

		// ensure partition is created
		if _, found := gocache.backers[partition]; !found {
			gocache.partitionMux.Lock()
			if _, found := gocache.backers[partition]; !found {
				gocache.backers[partition] = backer.New(backer.NoExpiration, defaultCacheScrapeMinutes*time.Minute)
			}
			gocache.partitionMux.Unlock()
		}

		// put in backing store key -> envelope
		gocache.backers[partition].Set(key, &envelope{
			message: response,
			time:    time.Now(),
		}, time.Duration(ttl)*time.Second)

		return true
	}

	return false
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
	key := gocache.key(request.Question)
	if "" == key {
		return nil, false
	}

	// no matching partition
	gocache.partitionMux.RLock()
	if _, found := gocache.backers[partition]; !found {
		gocache.partitionMux.RUnlock()
		return nil, false
	}
	gocache.partitionMux.RUnlock()

	value, found := gocache.backers[partition].Get(key)
	if !found {
		return nil, false
	}
	envelope := value.(*envelope)
	if envelope == nil || envelope.message == nil || util.IsEmptyResponse(envelope.message) {
		return nil, false
	}

	// use the time from the envelope to determine how long the message has been in the cache to adjust the ttl
	delta := time.Now().Sub(envelope.time)

	// copy the message to return it instead of the original
	messageCopy := envelope.message.Copy()

	// update message id to match request id
	messageCopy.MsgHdr.Id = request.MsgHdr.Id

	// count down/change ttl values in response
	secondDelta := uint32(delta / time.Second)
	adjustTtls(secondDelta, messageCopy.Answer)
	adjustTtls(secondDelta, messageCopy.Ns)
	adjustTtls(secondDelta, messageCopy.Extra)

	return messageCopy, true
}

func (gocache *gocache) Size() uint32 {
	count := uint32(0)
	gocache.partitionMux.RLock()
	for _, v := range gocache.backers {
		count += uint32(v.ItemCount())
	}
	gocache.partitionMux.RUnlock()
	return count
}

// delete all items from the cache
func (gocache *gocache) Clear() {
	gocache.partitionMux.Lock()
	for _, v := range gocache.backers {
		v.Flush()
	}
	gocache.partitionMux.Unlock()
}
