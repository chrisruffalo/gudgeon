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

// key delimeter
const (
	delimeter                 = "|"
	dnsMaxTTL                 = uint32(604800)
	defaultCacheScrapeMinutes = 1
)

type envelope struct {
	message *dns.Msg
	time    time.Time
}

type Cache interface {
	Store(partition string, request *dns.Msg, response *dns.Msg) bool
	Query(partition string, request *dns.Msg) (*dns.Msg, bool)
	Map() map[string]backer.Item
	Size() uint32
}

// keeps a pointer to the backer as well as a map of
// group -> int mappings that saves several bytes for
// each key entry. (this optimization may be overkill)
type gocache struct {
	backer          *backer.Cache
	partitionIdx    int
	partitionIdxMap map[string]int
	idMux           sync.Mutex
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

func New() Cache {
	gocache := new(gocache)
	gocache.backer = backer.New(backer.NoExpiration, defaultCacheScrapeMinutes*time.Minute)
	gocache.partitionIdx = 0
	gocache.partitionIdxMap = make(map[string]int, 0)
	return gocache
}

func minTTL(currentMin uint32, records []dns.RR) uint32 {
	for _, value := range records {
		currentMin = min(currentMin, value.Header().Ttl)
	}
	return currentMin
}

// make string key from partition + message
func (gocache *gocache) key(partition string, questions []dns.Question) string {
	// ensure the partition is lowercase
	partition = strings.ToLower(strings.TrimSpace(partition))

	// it is very likely, even at low query rates, that two threads are
	// hitting this section from the same group and so we have to tread lightly when
	// setting values in the map potentially at the same time
	if _, found := gocache.partitionIdxMap[partition]; !found {
		gocache.idMux.Lock()
		if _, found := gocache.partitionIdxMap[partition]; !found {
			gocache.partitionIdx++
			gocache.partitionIdxMap[partition] = gocache.partitionIdx
		}
		gocache.idMux.Unlock()
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%d", gocache.partitionIdxMap[partition]))
	if len(questions) > 0 {
		for _, question := range questions {
			builder.WriteString(delimeter)
			builder.WriteString(question.Name)
			builder.WriteString(delimeter)
			builder.WriteString(dns.Class(question.Qclass).String())
			builder.WriteString(delimeter)
			builder.WriteString(dns.Type(question.Qtype).String())
		}
	}
	return strings.ToLower(builder.String())
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
		key := gocache.key(partition, request.Question)
		if "" == key {
			return false
		}

		// copy response to envelope
		envelope := new(envelope)
		envelope.message = response
		envelope.time = time.Now()

		// put in backing store key -> envelope
		gocache.backer.Set(key, envelope, time.Duration(ttl)*time.Second)

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
	key := gocache.key(partition, request.Question)

	if "" == key {
		return nil, false
	}

	value, found := gocache.backer.Get(key)
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
	return uint32(gocache.backer.ItemCount())
}

func (gocache *gocache) Map() map[string]backer.Item {
	return gocache.backer.Items()
}
