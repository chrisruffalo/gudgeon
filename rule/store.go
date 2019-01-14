package rule

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
	Load(group string, rules []Rule) uint64
	IsMatch(group string, domain string) Match
	IsMatchAny(group []string, domain string) Match
}

// order of applying/creating/using rules
var ruleApplyOrder = []uint8{ALLOW, BLOCK}

// creates whatever gudgeon considers to be the default store
func CreateDefaultStore() RuleStore {
	return CreateStore("")
}

func CreateStore(backingStoreType string) RuleStore {

	// first create the complex rule store wrapper
	store := new(complexStore)
	store.complexRules = make(map[uint8]map[string][]Rule)
	for _, element := range ruleApplyOrder {
		store.complexRules[element] = make(map[string][]Rule)
	}

	// create appropriate backing store
	var backingStore RuleStore
	if "radix" == backingStoreType {
		backingStore = new(radixStore)
	} else {
		backingStore = new(memoryStore)
	}

	// set backing store
	store.backingStore = backingStore

	// finalize and return store
	return store
}
