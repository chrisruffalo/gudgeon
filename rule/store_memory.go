package rule

import (
	"sort"
	"strings"

	"github.com/akutz/sortfold"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type memoryStore struct {
	rules map[string][]string
}

func (store *memoryStore) Load(conf *config.GudgeonConfig, list *config.GudgeonList, rules []Rule) uint64 {
	// need some actual rules
	if len(rules) < 1 {
		return 0
	}

	if store.rules == nil {
		store.rules = make(map[string][]string)
	}

	// filter through rules and count how many rules are in use
	counter := uint64(0)
	for _, r := range rules {
		if r == nil {
			continue
		}
		counter++
	}
	// making the array this way saves memory for very large lists (500K - 1M+ lines)
	// and doesn't really take any more time
	idx := 0
	stringRules := make([]string, counter)
	for _, r := range rules {
		if r == nil {
			continue
		}
		stringRules[idx] = r.Text()
		idx++
	}

	// case insensitive string/rule sort
	sort.Slice(stringRules, func(i, j int) bool {
		return sortfold.CompareFold(stringRules[i], stringRules[j]) < 0
	})

	// save rules
	store.rules[list.CanonicalName()] = stringRules

	return counter
}

func (store *memoryStore) foundInList(rules []string, domain string) (bool, string) {
	// search for the domain
	idx := sort.Search(len(rules), func(i int) bool {
		return sortfold.CompareFold(rules[i], domain) >= 0
	})

	// check that search found what we expected and return true if found
	if idx < len(rules) && strings.EqualFold(rules[idx], domain) {
		return true, rules[idx]
	}

	// otherwise return false
	return false, ""
}

func (store *memoryStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {
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

	domains := util.DomainList(domain)

	for _, list := range allowLists {
		rules, found := store.rules[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(rules, d); found {
				return MatchAllow, list, ruleString
			}
		}
	}

	for _, list := range blockLists {
		rules, found := store.rules[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domains {
			if found, ruleString := store.foundInList(rules, d); found {
				return MatchBlock, list, ruleString
			}
		}
	}

	return MatchNone, nil, ""
}
