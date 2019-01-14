package rule

import (
	"strings"

	iradix "github.com/armon/go-radix"
	//"github.com/chrisruffalo/gudgeon/util"
)

type radixStore struct {
	tree     *iradix.Tree
	groupMap map[string]uint
	groupIdx uint
}

func (store *radixStore) Load(group string, rules []Rule) uint64 {
	if store.groupMap == nil {
		store.groupMap = make(map[string]uint, 0)
	}
	if _, found := store.groupMap[group]; !found {
		store.groupIdx++
		store.groupMap[group] = store.groupIdx
	}
	groupIdx := store.groupMap[group]

	if store.tree == nil {
		store.tree = iradix.New()
	}

	// categorize and put rules
	counter := uint64(0)
	for _, rule := range rules {
		// don't load nil rules
		if rule == nil {
			continue
		}

		// reverse rule
		text := strings.ToLower(rule.Text()) //util.ReverseDomainTree(strings.ToLower(rule.Text()))
		if "" == text {
			continue
		}

		// lookup rule
		value, found := store.tree.Get(text)
		var valueMap map[uint]uint8
		if !found {
			// insert new value holder structure
			valueMap = make(map[uint]uint8, 0)
			store.tree.Insert(text, valueMap)
		} else {
			valueMap = value.(map[uint]uint8)
		}

		// add rule group to map
		(valueMap)[groupIdx] = rule.RuleType()

		// increment counter
		counter++
	}

	return counter
}

func (store *radixStore) IsMatchAny(groups []string, domain string) Match {
	if "" == strings.TrimSpace(domain) {
		return MatchNone
	}

	// reverse domain
	reverse := strings.ToLower(domain) //strings.ToLower(util.ReverseDomainTree(domain))
	if "" == domain {
		return MatchNone
	}

	match := MatchNone

	check := reverse
	for len(check) > 0 {
		value, found := store.tree.Get(check)
		if found {
			valueMap := value.(map[uint]uint8)
			for _, group := range groups {
				if groupIdx, ok := store.groupMap[group]; ok {
					if ruleType, ok := (valueMap)[groupIdx]; ok {
						if ruleType == ALLOW {
							return MatchAllow
						} else {
							match = MatchBlock
						}
					}
				}
			}
		}

		split := strings.Split(check, ".")
		//check = strings.Join(split[:len(split) - 1], ".")
		check = strings.Join(split[1:], ".")
	}

	return match
}

// default implementation of IsMatch
func (store *radixStore) IsMatch(group string, domain string) Match {
	return store.IsMatchAny([]string{group}, domain)
}
