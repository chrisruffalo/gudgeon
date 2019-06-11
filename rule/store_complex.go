package rule

import (
	"github.com/chrisruffalo/gudgeon/config"
)

type complexStore struct {
	backingStore RuleStore
	complexRules map[string][]ComplexRule
}

func (store *complexStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.complexRules = make(map[string][]ComplexRule, 0)

	for _, list := range lists {
		if _, found := store.complexRules[list.CanonicalName()]; !found {
			store.complexRules[list.CanonicalName()] = make([]ComplexRule, 0)
		}
	}

	if store.backingStore != nil {
		store.backingStore.Init(sessionRoot, config, lists)
	}
}

func (store *complexStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {

}

func (store *complexStore) Load(list *config.GudgeonList, rule string) {
	// complex rules are locally stored
	var complexRule ComplexRule
	if IsComplex(rule) {
		complexRule = CreateComplexRule(rule)
		if complexRule != nil {
			store.complexRules[list.CanonicalName()] = append(store.complexRules[list.CanonicalName()], complexRule)
		}
	} else if store.backingStore != nil {
		store.backingStore.Load(list, rule)
	}
}

func (store *complexStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	// no-op

	if store.backingStore != nil {
		store.backingStore.Finalize(sessionRoot, lists)
	}
}

func (store *complexStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {
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

	for _, list := range allowLists {
		rules, found := store.complexRules[list.CanonicalName()]
		if !found {
			continue
		}

		for _, rule := range rules {
			if rule.IsMatch(domain) {
				return MatchAllow, list, rule.Text()
			}
		}
	}

	for _, list := range blockLists {
		rules, found := store.complexRules[list.CanonicalName()]
		if !found {
			continue
		}

		for _, rule := range rules {
			if rule.IsMatch(domain) {
				return MatchBlock, list, rule.Text()
			}
		}
	}

	// delegate to backing store if no result found
	if store.backingStore != nil {
		return store.backingStore.FindMatch(lists, domain)
	}

	return MatchNone, nil, ""
}

func (store *complexStore) Close() {
	// default no-op
}
