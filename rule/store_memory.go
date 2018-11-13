package rule

import (
	"strings"
	"sort"
	"github.com/chrisruffalo/gudgeon/util"
)

// map rule to group to list type to group names
type memoryStore struct {
	rules map[string]map[uint8]*[]string
}

func (store *memoryStore) Load(group string, rules []Rule) {
	if store.rules == nil {
		store.rules = make(map[string]map[uint8]*[]string)
	}

	// categorize and put rules
	for _, rule := range rules {
		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make(map[uint8]*[]string)
		}

		// if group list not present for rule type create group list
		ruleType := rule.RuleType()
		if store.rules[lower][ruleType] == nil {
			newEmptySlice := make([]string, 0)
			store.rules[lower][ruleType] = &newEmptySlice
		}

		// append group to list belonging to rule type
		if !util.StringIn(group, *store.rules[lower][ruleType]) {
			appendedSlice := append(*store.rules[lower][ruleType], group)
			// just sort it each time so we can search it later
			sort.Strings(appendedSlice)

			store.rules[lower][ruleType] = &appendedSlice
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
			if store.rules[domain][ruleType] == nil {
				continue
			}

			// get groups that this rule type applies to
			applyToGroups := *store.rules[domain][ruleType]

			// skip forward if rule type is not represented
			if len(applyToGroups) < 1 {
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
