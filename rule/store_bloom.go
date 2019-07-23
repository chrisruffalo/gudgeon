package rule

import (
	"github.com/willf/bloom"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	defaultRuleCount = uint(1000) // just 10000 in the default count, for when the lines in the file can't be counted
	bloomRate        = 0.5        // when the **list is full** a 0.5% chance of false positive (which can be mitigated by another store)
)

type bloomStore struct {
	baseStore

	blooms           map[string]*bloom.BloomFilter
	backingStore     Store
	defaultRuleCount uint
}

func (store *bloomStore) createBloom(config *config.GudgeonConfig, list *config.GudgeonList) {
	if _, found := store.blooms[list.CanonicalName()]; !found {
		// get lines in file
		var err error
		linesInFile := store.defaultRuleCount
		if config != nil {
			linesInFile, err = util.LineCount(config.PathToList(list))
			if err != nil {
				linesInFile = store.defaultRuleCount
			}
		}
		store.blooms[list.CanonicalName()] = bloom.NewWithEstimates(linesInFile, bloomRate)
	}
}

func (store *bloomStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.blooms = make(map[string]*bloom.BloomFilter)
	if store.defaultRuleCount <= 0 {
		store.defaultRuleCount = defaultRuleCount
	}

	for _, list := range lists {
		store.createBloom(config, list)
	}

	if store.backingStore != nil {
		store.backingStore.Init(sessionRoot, config, lists)
	}
}

func (store *bloomStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	store.createBloom(config, list)
	store.removeList(list)

	if store.backingStore != nil {
		store.backingStore.Clear(config, list)
	}
}

func (store *bloomStore) Load(list *config.GudgeonList, rule string) {
	// add to filter only when not in the filter already
	if !store.blooms[list.CanonicalName()].TestString(rule) {
		store.blooms[list.CanonicalName()].AddString(rule)
	}
	store.addList(list)

	if store.backingStore != nil {
		store.backingStore.Load(list, rule)
	}
}

func (store *bloomStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {

	if store.backingStore != nil {
		store.backingStore.Finalize(sessionRoot, lists)
	}
}

func (store *bloomStore) foundInList(filter *bloom.BloomFilter, domain string) (bool, string) {
	// otherwise return false
	return filter.TestString(domain), ""
}

func (store *bloomStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {

	domains := util.DomainList(domain)

	match, list, rule := store.matchForEachOfTypeIn(config.ALLOW, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		filter, found := store.blooms[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(filter, d); found {
				if store.backingStore != nil {
					return store.backingStore.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchAllow, list, ruleString
			}
		}
		return MatchNone, nil, ""
	})

	if MatchNone != match {
		return match, list, rule
	}

	match, list, rule = store.matchForEachOfTypeIn(config.BLOCK, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		filter, found := store.blooms[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(filter, d); found {
				if store.backingStore != nil {
					return store.backingStore.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchBlock, list, ruleString
			}
		}
		return MatchNone, nil, ""
	})

	return match, list, rule
}

func (store *bloomStore) Close() {
	// remove reference to blooms
	store.blooms = make(map[string]*bloom.BloomFilter)

	if store.backingStore != nil {
		store.backingStore.Close()
	}
}
