package rule

import (
	"github.com/chrisruffalo/gudgeon/config"
)

type complexStore struct {
	backingStore RuleStore
	complexRules map[string][]Rule
}

func (store *complexStore) Load(conf *config.GudgeonConfig, list *config.GudgeonList, sessionRoot string, rules []Rule) uint64 {
	if store.complexRules == nil {
		store.complexRules = make(map[string][]Rule, 0)
	}
	if _, found := store.complexRules[list.CanonicalName()]; !found {
		store.complexRules[list.CanonicalName()] = make([]Rule, 0)
	}

	// make decisions based on the rule type
	counter := uint64(0)
	for idx, rule := range rules {
		if rule == nil {
			continue
		}

		// complex rules are locally stored
		if rule.IsComplex() {
			store.complexRules[list.CanonicalName()] = append(store.complexRules[list.CanonicalName()], rule)

			// rule is nilled out from list forwarded to next component
			rules[idx] = nil

			// add to rule counter for return
			counter++
		}
	}

	// backing store load is handled
	if store.backingStore != nil {
		return counter + store.backingStore.Load(conf, list, sessionRoot, rules)
	}

	return counter
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
