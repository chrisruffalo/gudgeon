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
}

func (store *bloomStore) Load(list *config.GudgeonList, rule string) {
	// add to filter only when not in the filter already
	if !store.blooms[list.CanonicalName()].TestString(rule) {
		store.blooms[list.CanonicalName()].AddString(rule)
	}

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
	// allow and block split
	allowLists := make([]*config.GudgeonList, 0)
	blockLists := make([]*config.GudgeonList, 0)
	for _, l := range lists {
		if ParseType(l.Type) == ALLOW {
			allowLists = append(allowLists, l)
		} else {
			blockLists = append(blockLists, l)
		}
	}

	domains := util.DomainList(domain)

	for _, list := range allowLists {
		filter, found := store.blooms[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(filter, d); found {
				if store.backingStore != nil {
					return store.backingStore.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchAllow, list, ruleString
			}
		}
	}

	for _, list := range blockLists {
		filter, found := store.blooms[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(filter, d); found {
				if store.backingStore != nil {
					return store.backingStore.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchBlock, list, ruleString
			}
		}
	}

	return MatchNone, nil, ""
}

func (store *bloomStore) Close() {
	// remove reference to blooms
	store.blooms = make(map[string]*bloom.BloomFilter)

	if store.backingStore != nil {
		store.backingStore.Close()
	}
}
