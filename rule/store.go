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
	Load(group string, rules []Rule, conf *config.GudgeonConfig, list *config.GudgeonList) uint64
	IsMatch(group string, domain string) Match
	IsMatchAny(group []string, domain string) Match
}

// order of applying/creating/using rules
var ruleApplyOrder = []uint8{ALLOW, BLOCK}

// creates whatever gudgeon considers to be the default store
func CreateDefaultStore() RuleStore {
	return CreateStore("bloom")
}

func CreateStore(backingStoreType string) RuleStore {

	// first create the complex rule store wrapper
	store := new(complexStore)
	store.complexRules = make(map[uint8]map[string][]Rule)
	for _, element := range ruleApplyOrder {
		store.complexRules[element] = make(map[string][]Rule)
	}

	// create appropriate backing store
	var delegate RuleStore
	if "radix" == backingStoreType {
		delegate = new(radixStore)
	} else if "bloom" == backingStoreType {
		delegate = new(bloomStore)
	} else {
		delegate = new(memoryStore)
	}

	// set backing store
	store.backingStore = delegate

	// finalize and return store
	return store
}
