package rule

import (
	"strings"

	"github.com/chrisruffalo/gudgeon/util"
)

// map rule to group to rule type
type memoryStore struct {
	rules map[string]map[string]*uint8
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[string]map[string]*uint8)
	}

	// categorize and put rules
	for _, rule := range rules {
		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make(map[string]*uint8)
		}

		// if rule not already allowed in this group set allowance
		if store.rules[lower][group] == nil || *store.rules[lower][group] != ALLOW {
			ruleType := rule.RuleType()
			store.rules[lower][group] = &ruleType
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
	root := util.RootDomain(domain)

	if store.rules[domain] != nil {
		for _, group := range groups {
			if store.rules[domain][group] != nil && *store.rules[domain][group] == ALLOW {
				return false
			} else if store.rules[domain][group] != nil && *store.rules[domain][group] == BLOCK {
				return true
			}
		}
	}

	if store.rules[root] != nil {
		for _, group := range groups {
			// get root domain
			if store.rules[root][group] != nil && *store.rules[root][group] == ALLOW {
				return false
			} else if store.rules[root][group] != nil && *store.rules[root][group] == BLOCK {
				return true
			}
		}
	}

	// check root domain after domain
	return false
}

// default implementation of IsMatch
func (store *memoryStore) IsMatch(group string, domain string) bool {
	return store.IsMatchAny([]string{group}, domain)
}
