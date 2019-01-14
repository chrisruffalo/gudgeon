package rule

import (
	"strings"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type memoryStore struct {
	rules    map[string]map[uint]bool
	groupMap map[string]uint
	groupIdx uint
}

func (store *memoryStore) Load(group string, rules []Rule, conf *config.GudgeonConfig, list *config.GudgeonList) uint64 {
	if store.groupMap == nil {
		store.groupMap = make(map[string]uint, 0)
	}
	if _, found := store.groupMap[group]; !found {
		store.groupIdx++
		store.groupMap[group] = store.groupIdx
	}
	groupIdx := store.groupMap[group]

	if store.rules == nil {
		store.rules = make(map[string]map[uint]bool)
	}

	// categorize and put rules
	counter := uint64(0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}

		lower := strings.ToLower(rule.Text())
		if store.rules[lower] == nil {
			store.rules[lower] = make(map[uint]bool)
		}

		// you can't overwrite an ALLOW because that takes precedence
		if !store.rules[lower][groupIdx] {
			store.rules[lower][groupIdx] = ALLOW == rule.RuleType()
			counter++
		}
	}

	return counter
}

func (store *memoryStore) IsMatchAny(groups []string, domain string) Match {
	// if we don't know about rules exit
	if store.rules == nil {
		return MatchNone
	}

	if "" == domain {
		return MatchNone
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
			groupIdx := store.groupMap[group]
			val, found := store.rules[domain][groupIdx]
			if found && val {
				return MatchAllow
				// otherwise if a value was found it must be false
			} else if found {
				blocked = true
			}
		}

		if blocked {
			return MatchBlock
		}
	}

	// process root domain if it is different
	sub := util.SubDomain(domain)
	if domain != sub {
		return store.IsMatchAny(groups, sub)
	}

	// no match
	return MatchNone
}

// default implementation of IsMatch
func (store *memoryStore) IsMatch(group string, domain string) Match {
	return store.IsMatchAny([]string{group}, domain)
}
