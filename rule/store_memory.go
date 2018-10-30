package rule

import (
	"sort"
	"github.com/chrisruffalo/gudgeon/util"
)

type memoryStore struct {
	rules map[uint8]map[string][]string
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[uint8]map[string][]string)
	}

	// categorize and put rules
	for _, rule := range rules {
		if store.rules[rule.RuleType()] == nil {
			store.rules[rule.RuleType()] = make(map[string][]string)
		}
		if store.rules[rule.RuleType()][group] == nil {
			store.rules[rule.RuleType()][group] = make([]string, 0)
		}
		store.rules[rule.RuleType()][group] = append(store.rules[rule.RuleType()][group], rule.Text())
	}

	// sort each list of rules based on type and group
	for ruleKey, _ := range store.rules {
		// now handle each group in map
		for groupKey, _ := range store.rules[ruleKey] {
			// sort the string
			sort.Strings(store.rules[ruleKey][groupKey])
		}
	}
}

func (store *memoryStore) IsMatch(group string, domain string) bool {
	// if we don't know about rules exit
	if store.rules == nil {
		return false
	}

	// go through each rule type and apply the rules in order
	for _, ruleType := range ruleApplyOrder {
		// continue if there are no rules of that type
		if store.rules[ruleType] == nil {
			continue
		}
		// look for the group of that rule type
		if store.rules[ruleType][group] == nil || len(store.rules[ruleType][group]) < 1 {
			continue
		}
		// get the list into a local slice for brevity
		memlist := store.rules[ruleType][group]
		// now try and find the item
		foundex := sort.SearchStrings(memlist, domain)
		if foundex < 0 || foundex >= len(memlist) {
			continue
		}
		result := memlist[foundex] == domain
		if !result {
			rootdomain := util.RootDomain(domain)
			foundex = sort.SearchStrings(memlist, rootdomain)
			if foundex >= 0 || foundex < len(memlist) {
				result = memlist[foundex] == rootdomain
			}
		}
		// create result
		if result {
			// whitelist is an immediate short circuit no/false
			if ruleType == WHITELIST {
				return false
			} else {
				// otherwise return true on match
				return true
			}
		}
	}

	// if nothing was ever found return false
	return false
}