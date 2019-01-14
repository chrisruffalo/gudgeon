package rule

import (
	"github.com/willf/bloom"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	bloomRate = 0.001 // basically a 0.1% chance of a false positive from the filter, which would necessitate going to another source to check
)

type bloomStore struct {
	groupAllowMap   map[string]*[]*config.GudgeonList // a list that defines what allow lists belong to the given group
	groupBlockMap   map[string]*[]*config.GudgeonList // a list that defines what block lists belong to the given group
	backingStoreMap map[string]*RuleStore             // if we want to do more concrete checking forward to a backing store, per list
	bloomFilters    map[string]*bloom.BloomFilter     // map list to filter
}

func (store *bloomStore) Load(group string, rules []Rule, conf *config.GudgeonConfig, list *config.GudgeonList) uint64 {
	// lazy make
	if store.groupAllowMap == nil {
		store.groupAllowMap = make(map[string]*[]*config.GudgeonList, 0)
	}
	if store.groupBlockMap == nil {
		store.groupBlockMap = make(map[string]*[]*config.GudgeonList, 0)
	}
	if store.backingStoreMap == nil {
		store.backingStoreMap = make(map[string]*RuleStore, 0)
	}
	if store.bloomFilters == nil {
		store.bloomFilters = make(map[string]*bloom.BloomFilter, 0)
	}

	currentMap := &store.groupBlockMap
	if ParseType(list.Type) == ALLOW {
		currentMap = &store.groupAllowMap
	}

	// get list of lists and create structure if it doesn't exist
	groupListMap, found := (*currentMap)[group]
	if !found {
		concreteMap := make([]*config.GudgeonList, 0)
		groupListMap = &concreteMap
		(*currentMap)[group] = groupListMap
	}
	inMap := false
	for _, mapList := range *groupListMap {
		if mapList.CanonicalName() == list.CanonicalName() {
			inMap = true
			break
		}
	}
	if !inMap {
		*groupListMap = append(*groupListMap, list)
	}

	// look for bloom filter and create it if it isn't found
	filter, filterFound := store.bloomFilters[list.CanonicalName()]
	if !filterFound {
		filter = bloom.NewWithEstimates(uint(len(rules)), bloomRate)
		store.bloomFilters[list.CanonicalName()] = filter
	}

	// load items into filter
	counter := uint64(0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		// only add rule if it isn't in the list already
		if !filter.TestString(rule.Text()) {
			filter.AddString(rule.Text())
			counter++
		}
	}

	return counter
}

func (store *bloomStore) IsMatchAny(groups []string, domain string) Match {
	// get list of domains that should be checked
	domains := util.DomainList(domain)

	allowLists := make([]*config.GudgeonList, 0)
	blockLists := make([]*config.GudgeonList, 0)
	for _, g := range groups {
		if _, found := store.groupAllowMap[g]; found {
			allowLists = append(allowLists, *(store.groupAllowMap[g])...)
		}

		if _, found := store.groupBlockMap[g]; found {
			blockLists = append(blockLists, *(store.groupBlockMap[g])...)
		}
	}

	for _, list := range allowLists {
		filter := store.bloomFilters[list.CanonicalName()]
		for _, c := range domains {
			if filter.TestString(c) {
				return MatchAllow
			}
		}
	}

	for _, list := range blockLists {
		filter := store.bloomFilters[list.CanonicalName()]
		for _, c := range domains {
			if filter.TestString(c) {
				return MatchBlock
			}
		}
	}

	return MatchNone
}

// default implementation of IsMatch
func (store *bloomStore) IsMatch(group string, domain string) Match {
	return store.IsMatchAny([]string{group}, domain)
}
