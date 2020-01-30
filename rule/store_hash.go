package rule

import (
	"strings"

	"github.com/twmb/murmur3"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type hashStore struct {
	baseStore

	hashes map[string]map[uint64]struct{}

	delegate Store
}

func (store *hashStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.hashes = make(map[string]map[uint64]struct{})
	for _, list := range lists {
		if _, found := store.hashes[list.CanonicalName()]; !found {
			store.hashes[list.CanonicalName()] = make(map[uint64]struct{})
		}
	}

	if store.delegate != nil {
		store.delegate.Init(sessionRoot, config, lists)
	}
}

func (store *hashStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	store.hashes[list.CanonicalName()] = make(map[uint64]struct{})
	store.removeList(list)

	if store.delegate != nil {
		store.delegate.Clear(config, list)
	}
}

func (store *hashStore) Load(list *config.GudgeonList, rule string) {
	store.hashes[list.CanonicalName()][murmur3.StringSum64(strings.ToLower(rule))] = struct{}{}
	store.addList(list)

	if store.delegate != nil {
		store.delegate.Load(list, rule)
	}
}

func (store *hashStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	if store.delegate != nil {
		store.delegate.Finalize(sessionRoot, lists)
	}
}

func (store *hashStore) foundInList(rules map[uint64]struct{}, domainHash uint64) bool {
	if _, found := rules[domainHash]; found {
		return true
	}

	// otherwise return false
	return false
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
		for idx := 0; idx < len(domainHashes); idx++ {
			if found := store.foundInList(rules, domainHashes[idx]); found  {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchAllow, list, domains[idx]
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
		for idx := 0; idx < len(domainHashes); idx++ {
			if found := store.foundInList(rules, domainHashes[idx]); found  {
				if store.delegate != nil {
					return store.delegate.FindMatch([]*config.GudgeonList{list}, domain)
				}
				return MatchBlock, list, domains[idx]
			}
		}
		return MatchNone, nil, ""
	})

	return match, list, rule
}

func (store *hashStore) Close() {
	// overwrite map with empty map
	store.hashes = make(map[string]map[uint64]struct{})

	if store.delegate != nil {
		store.delegate.Close()
	}
}
