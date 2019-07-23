package rule

import (
	"github.com/chrisruffalo/gudgeon/config"
)

type complexStore struct {
	baseStore

	backingStore Store
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
	store.complexRules[list.CanonicalName()] = make([]ComplexRule, 0)
	store.removeList(list)

	if store.backingStore != nil {
		store.backingStore.Clear(config, list)
	}
}

func (store *complexStore) Load(list *config.GudgeonList, rule string) {
	// complex rules are locally stored
	var complexRule ComplexRule
	if list.Regex != nil && *list.Regex {
		complexRule = specifyRegexOnlyRule(rule)
		if complexRule != nil {
			store.complexRules[list.CanonicalName()] = append(store.complexRules[list.CanonicalName()], complexRule)
		}
	} else if IsComplex(rule) {
		complexRule = createComplexRule(rule)
		if complexRule != nil {
			store.complexRules[list.CanonicalName()] = append(store.complexRules[list.CanonicalName()], complexRule)
		}
	} else if store.backingStore != nil {
		store.backingStore.Load(list, rule)
	}
	store.addList(list)
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
		if l.ParsedType() == config.ALLOW {
			allowLists = append(allowLists, l)
		} else {
			blockLists = append(blockLists, l)
		}
	}

	match, list, rule := store.matchForEachOfTypeIn(config.ALLOW, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		rules, found := store.complexRules[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}

		for _, rule := range rules {
			if rule.IsMatch(domain) {
				return MatchAllow, list, rule.Text()
			}
		}
		return MatchNone, nil, ""
	})

	if MatchNone != match {
		return match, list, rule
	}

	match, list, rule = store.matchForEachOfTypeIn(config.BLOCK, lists, func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string) {
		rules, found := store.complexRules[list.CanonicalName()]
		if !found {
			return MatchNone, nil, ""
		}

		for _, rule := range rules {
			if rule.IsMatch(domain) {
				return MatchBlock, list, rule.Text()
			}
		}
		return MatchNone, nil, ""
	})

	if MatchNone != match {
		return match, list, rule
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
