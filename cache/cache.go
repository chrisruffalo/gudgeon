package cache

import (
	"time"

	"github.com/miekg/dns"
	backer "github.com/patrickmn/go-cache"
)

// key delimeter
const delimeter = "|"

type entry struct {
	answers []dns.RR
	ns []dns.RR
	extra []dns.RR
}

type Cache interface {
	Store(group string, msg *dns.Msg)
	Query(group string, msg *dns.Msg) bool
}

type gocache struct {
	backer *backer.Cache
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

func (gocache *gocache) Store(group string, msg *dns.Msg) {
	// create copy of message
	mCopy := msg.Copy()

	// create key from message
	key := key(group, mCopy.Question)
	if "" == key {
		return
	}

	// store parts
	entry := new(entry)
	entry.answers = make([]dns.RR, len(mCopy.Answer))
	for idx, a := range mCopy.Answer {
		entry.answers[idx] = a
	}
	entry.ns = make([]dns.RR, len(mCopy.Ns))
	for idx, ns := range mCopy.Ns {
		entry.ns[idx] = ns
	}
	entry.extra = make([]dns.RR, len(mCopy.Extra))
	for idx, ex := range mCopy.Extra {
		entry.extra[idx] = ex
	}

	// stuff in backing store
	gocache.backer.Set(key, entry, backer.NoExpiration)
}

func (gocache *gocache) Query(group string, msg *dns.Msg) bool {
	// get key
	key := key(group, msg.Question)
	if "" == key {
		return false
	}

	value, found := gocache.backer.Get(key)
	if !found {
		return false
	}
	entry := value.(*entry)

	msg.Answer = make([]dns.RR, len(entry.answers))
	for idx, value := range entry.answers {
		msg.Answer[idx] = value
	}
	msg.Ns = make([]dns.RR, len(entry.ns))
	for idx, value := range entry.ns {
		msg.Ns[idx] = value
	}
	msg.Extra = make([]dns.RR, len(entry.extra)) 
	for idx, value := range entry.extra {
		msg.Extra[idx] = value
	}


	return true
}