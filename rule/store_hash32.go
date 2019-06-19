package rule

import (
	"fmt"
	"sort"
	"strings"

	"github.com/twmb/murmur3"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type hashStore32 struct {
	hashes map[string][]uint32

	delegate RuleStore
}

func (store *hashStore32) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.hashes = make(map[string][]uint32)
	for _, list := range lists {
		if _, found := store.hashes[list.CanonicalName()]; !found {
			startingArrayLength := uint(0)
			if config != nil {
				startingArrayLength, _ = util.LineCount(config.PathToList(list))
			}
			store.hashes[list.CanonicalName()] = make([]uint32, 0, startingArrayLength)
		}
	}

	if store.delegate != nil {
		store.delegate.Init(sessionRoot, config, lists)
	}
}

func (store *hashStore32) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	startingArrayLength := uint(0)
	if config != nil {
		startingArrayLength, _ = util.LineCount(config.PathToList(list))
	}
	store.hashes[list.CanonicalName()] = make([]uint32, 0, startingArrayLength)
}

func (store *hashStore32) Load(list *config.GudgeonList, rule string) {
	store.hashes[list.CanonicalName()] = append(store.hashes[list.CanonicalName()], murmur3.StringSum32(strings.ToLower(rule)))

	if store.delegate != nil {
		store.delegate.Load(list, rule)
	}
}

func (store *hashStore32) Finalize(sessionRoot string, lists []*config.GudgeonList) {
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

func (store *hashStore32) foundInList(rules []uint32, domainHash uint32) (bool, uint32) {
	// search for the domain
	idx := sort.Search(len(rules), func(i int) bool {
		return rules[i] >= domainHash
	})

	// check that search found what we expected and return true if found
	if idx < len(rules) && rules[idx] == domainHash {
		return true, rules[idx]
	}

	// otherwise return false
	return false, uint32(0)
}

func (store *hashStore32) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {

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

	// get domain hashes
	domains := util.DomainList(domain)
	domainHashes := make([]uint32, len(domains))
	for idx, d := range domains {
		domainHashes[idx] = murmur3.StringSum32(strings.ToLower(d))
	}

	for _, list := range allowLists {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchAllow, list, fmt.Sprintf("%d", ruleHash)
			}
		}
	}

	for _, list := range blockLists {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchBlock, list, fmt.Sprintf("%d", ruleHash)
			}
		}
	}

	return MatchNone, nil, ""
}

func (store *hashStore32) Close() {
	// overwrite map with empty map
	store.hashes = make(map[string][]uint32)

	if store.delegate != nil {
		store.delegate.Close()
	}
}
