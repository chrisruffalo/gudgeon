package rule

import (
	"github.com/chrisruffalo/gudgeon/config"
)

// a match can be:
// allow (don't block, override/bypass block)
// block (explicit block)
// none (no reason found to block or allow)
type Match uint8

const (
	MatchAllow Match = 2
	MatchBlock Match = 1
	MatchNone  Match = 0
)

type RuleStore interface {
	Load(conf *config.GudgeonConfig, list *config.GudgeonList, rules []Rule) uint64
	FindMatch(lists []*config.GudgeonList, domain string) Match
}

// order of applying/creating/using rules
var ruleApplyOrder = []uint8{ALLOW, BLOCK}

// creates whatever gudgeon considers to be the default store
func CreateDefaultStore() RuleStore {
	return CreateStore("mem")
}

func CreateStore(backingStoreType string) RuleStore {
	// first create the complex rule store wrapper
	store := new(complexStore)

	// create appropriate backing store
	var delegate RuleStore
	delegate = new(memoryStore)

	// set backing store
	store.backingStore = delegate

	// finalize and return store
	return store
}
