package rule

import (
	"fmt"
	"sort"
	"strings"

	"github.com/twmb/murmur3"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

type hashStore32 struct {
	hashes map[string][]uint32
}

func (store *hashStore32) Load(conf *config.GudgeonConfig, list *config.GudgeonList, sessionRoot string, rules []Rule) uint64 {
	if store.hashes == nil {
		store.hashes = make(map[string][]uint32)
	}

	// filter through rules and count how many rules are in use
	counter := uint64(0)
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		counter++
	}

	idx := 0
	hashRules := make([]uint32, counter)

	// evaluate each rule
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		// hash lower case rule
		hashRules[idx] = murmur3.StringSum32(strings.ToLower(rule.Text()))
		idx++
	}

	// sort
	sort.Slice(hashRules, func(i, j int) bool {
		return hashRules[i] < hashRules[j]
	})

	store.hashes[list.CanonicalName()] = hashRules

	return counter
}

func (store *hashStore32) foundInList(rules []uint32, domainHash uint32) (bool, uint32) {
	// search for the domain
	idx := sort.Search(len(rules), func(i int) bool {
		return rules[i] >= domainHash
	})

	// check that search found what we expected and return true if found
	if idx < len(rules) && rules[idx] == domainHash {
		return true, rules[idx]
	}

	// otherwise return false
	return false, uint32(0)
}

func (store *hashStore32) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {

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

	// get domain hashes
	domains := util.DomainList(domain)
	domainHashes := make([]uint32, len(domains))
	for idx, d := range domains {
		domainHashes[idx] = murmur3.StringSum32(strings.ToLower(d))
	}

	for _, list := range allowLists {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				return MatchAllow, list, fmt.Sprintf("%d", ruleHash)
			}
		}
	}

	for _, list := range blockLists {
		rules, found := store.hashes[list.CanonicalName()]
		if !found {
			continue
		}
		for _, d := range domainHashes {
			if found, ruleHash := store.foundInList(rules, d); found && ruleHash > 0 {
				return MatchBlock, list, fmt.Sprintf("%d", ruleHash)
			}
		}
	}

	return MatchNone, nil, ""
}
