package rule

import (
	"strings"
	"sort"
	"github.com/chrisruffalo/gudgeon/util"
)

// map rule to allowed types (0 or 1) and then to groups for that allowed type
type memoryStore struct {
	rules map[string][][]string
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[string][][]string)
	}

	// categorize and put rules
	for _, rule := range rules {
		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make([][]string, 2)
		}

		// if group list not present for rule type create group list
		ruleType := rule.RuleType()
		if store.rules[lower][ruleType] == nil {
			store.rules[lower][ruleType] = make([]string, 0)
		}

		// append group to list belonging to rule type
		if !util.StringIn(group, store.rules[lower][ruleType]) {
			store.rules[lower][ruleType] = append(store.rules[lower][ruleType], group)
			// just sort it each time so we can search it later
			sort.Strings(store.rules[lower][ruleType])
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
		// do the same thing for each rule type
		for _, ruleType := range ruleApplyOrder {
			// get groups that this rule type applies to
			applyToGroups := store.rules[domain][ruleType]

			// skip forward if rule type is not represented
			if applyToGroups == nil || len(applyToGroups) < 1 {
				continue
			}

			// look for group in list of groups
			for _, group := range groups {
				// get index for search
				idx := sort.SearchStrings(applyToGroups, group)

				// if found 
				if idx >= 0 && idx < len(applyToGroups) && applyToGroups[idx] == group {
					if ruleType == ALLOW {
						return false
					} else {
						return true
					}
				}
			}
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
