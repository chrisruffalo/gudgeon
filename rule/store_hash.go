package rule

import (
	"fmt"
	"sort"
	"strings"

	"github.com/twmb/murmur3"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type hashStore struct {
	baseStore

	hashes map[string][]uint64

	delegate Store
}

func (store *hashStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.hashes = make(map[string][]uint64)
	for _, list := range lists {
		if _, found := store.hashes[list.CanonicalName()]; !found {
			startingArrayLength := uint(0)
			if config != nil {
				startingArrayLength, _ = util.LineCount(config.PathToList(list))
			}
			store.hashes[list.CanonicalName()] = make([]uint64, 0, startingArrayLength)
		}
	}

	if store.delegate != nil {
		store.delegate.Init(sessionRoot, config, lists)
	}
}

func (store *hashStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	startingArrayLength := uint(0)
	if config != nil {
		startingArrayLength, _ = util.LineCount(config.PathToList(list))
	}
	store.hashes[list.CanonicalName()] = make([]uint64, 0, startingArrayLength)
	store.removeList(list)

	if store.delegate != nil {
		store.delegate.Clear(config, list)
	}
}

func (store *hashStore) Load(list *config.GudgeonList, rule string) {
	store.hashes[list.CanonicalName()] = append(store.hashes[list.CanonicalName()], murmur3.StringSum64(strings.ToLower(rule)))
	store.addList(list)

	if store.delegate != nil {
		store.delegate.Load(list, rule)
	}
}

func (store *hashStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	for k := range store.hashes {
		// sort
		sort.Slice(store.hashes[k], func(i, j int) bool {
			return store.hashes[k][i] < store.hashes[k][j]
		})
	}

	if store.delegate != nil {
		store.delegate.Finalize(sessionRoot, lists)
	}
}

func (store *hashStore) foundInList(rules []uint64, domainHash uint64) (bool, uint64) {
	// search for the domain
	idx := sort.Search(len(rules), func(i int) bool {
		return rules[i] >= domainHash
	})

	// check that search found what we expected and return true if found
	if idx < len(rules) && rules[idx] == domainHash {
		return true, rules[idx]
	}

	// otherwise return false
	return false, uint64(0)
}

func (store *hashStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {

	// get domain hashes
	domains := util.DomainList(domain)
	domainHashes := make([]uint64, len(domains))
	for idx, d := range domains {
		domainHashes[idx] = murmur3.StringSum64(strings.ToLower(d))
	}

	match, list, rule := store.matchForEachOfTypeIn(config.ALLOW, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchAllow, list, fmt.Sprintf("%d", ruleHash)
			}
		}
		return MatchNone, nil, ""
	})

	if MatchNone != match {
		return match, list, rule
	}

	match, list, rule = store.matchForEachOfTypeIn(config.BLOCK, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchBlock, list, fmt.Sprintf("%d", ruleHash)
			}
		}
		return MatchNone, nil, ""
	})

	return match, list, rule
}

func (store *hashStore) Close() {
	// overwrite map with empty map
	store.hashes = make(map[string][]uint64)

	if store.delegate != nil {
		store.delegate.Close()
	}
}
