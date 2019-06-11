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

func (store *memoryStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	store.rules = make(map[string][]string)

	for _, list := range lists {
		if _, found := store.rules[list.CanonicalName()]; !found {
			startingArrayLength := uint(0)
			if config != nil {
				startingArrayLength, _ = util.LineCount(config.PathToList(list))
			}
			store.rules[list.CanonicalName()] = make([]string, 0, startingArrayLength)
		}
	}
}

func (store *memoryStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	startingArrayLength := uint(0)
	if config != nil {
		startingArrayLength, _ = util.LineCount(config.PathToList(list))
	}
	store.rules[list.CanonicalName()] = make([]string, 0, startingArrayLength)
}

func (store *memoryStore) Load(list *config.GudgeonList, rule string) {
	store.rules[list.CanonicalName()] = append(store.rules[list.CanonicalName()], strings.ToLower(rule))
}

func (store *memoryStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	for _, list := range lists {
		// case insensitive string/rule sort
		sort.Slice(store.rules[list.CanonicalName()], func(i, j int) bool {
			return sortfold.CompareFold(store.rules[list.CanonicalName()][i], store.rules[list.CanonicalName()][j]) < 0
		})
	}
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

func (store *memoryStore) Close() {
	// default no-op
}
