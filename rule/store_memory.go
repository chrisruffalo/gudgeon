package rule

import (
	"strings"
	"github.com/chrisruffalo/gudgeon/util"
)


type memoryStore struct {
	rules map[string]map[string]uint8
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[string]map[string]uint8)
	}

	// categorize and put rules
	for _, rule := range rules {
		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make(map[string]uint8)
		}

		// if group list not present for rule type create group list
		ruleType := rule.RuleType()
		if ALLOW != store.rules[lower][group] {
			store.rules[lower][group] = ruleType
		}
	}
}

func (store *memoryStore) IsMatchAny(groups []string, domain string) bool {
	// if we don't know about rules exit
	if store.rules == nil {
		return false
	}

	if "" == domain {
		return false
	}

	// domain matching is done on lower case domains
	domain = strings.ToLower(domain)
	if store.rules[domain] != nil {
		ruleType := uint8(99)

		for _, group := range groups {
			if ruleType == ALLOW {
				break
			}
			ruleType = store.rules[domain][group]
		}

		if ruleType == ALLOW {
			return false
		} else if ruleType == BLOCK {
			return true
		}
	}

	// process root domain if it is different
	root := util.RootDomain(domain)
	if domain != root {
		return store.IsMatchAny(groups, root)
	}
	
	// check root domain after domain
	return false
}

// default implementation of IsMatch
func (store *memoryStore) IsMatch(group string, domain string) bool {
	return store.IsMatchAny([]string{group}, domain)
}
