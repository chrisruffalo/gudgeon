package rule

import (
	"strings"
	"github.com/chrisruffalo/gudgeon/util"
)


type memoryStore struct {
	rules map[string]map[string]bool
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[string]map[string]bool)
	}

	// categorize and put rules
	for _, rule := range rules {
		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make(map[string]bool)
		}

		// you can't overwrite an ALLOW because that takes precedence
		if !store.rules[lower][group] {
			store.rules[lower][group] = ALLOW == rule.RuleType()
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
		// value for when all the groups are checked
		// and when a group is found but not TRUE/ALLOW
		// then it needs to be blocked
		blocked := false

		// go through each group
		for _, group := range groups {
			// if any group has ALLOW (true) as a value
			// then return false immediately according to
			// the whitelist behavior
			val, found := store.rules[domain][group]
			if found && val {
				return false
			// otherwise if a value was found it must be false
			} else if found {
				blocked = true
			}
		}

		return blocked
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
