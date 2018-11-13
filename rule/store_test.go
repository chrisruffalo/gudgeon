package rule

import (
	"testing"

	"github.com/chrisruffalo/gudgeon/testutil"
)

const (
	// one million rules
	benchRules = 1000000
)

type ruleList struct {
	group     string
	rule      string
	ruleType  uint8
	matching  []string
	nomatches []string
}

type ruleStoreCreator func() RuleStore

func testStore(ruleData []ruleList, createRuleStore ruleStoreCreator, t *testing.T) {

	for _, data := range ruleData {
		// create rule and rule list
		rule := CreateRule(data.rule, data.ruleType)
		rules := []Rule{rule}

		// load rules into target store
		store := createRuleStore()
		store.Load(data.group, rules)

		// check matching
		for _, expectedMatch := range data.matching {
			if !store.IsMatch(data.group, expectedMatch) {
				t.Errorf("Rule '%s' of type %d expected to match '%s' but did not", data.rule, data.ruleType, expectedMatch)
			}
		}

		// check non-matching
		for _, noMatch := range data.nomatches {
			if store.IsMatch(data.group, noMatch) {
				t.Errorf("Rule '%s' of type %d not expected to match '%s' but did", data.rule, data.ruleType, noMatch)
			}
		}
	}
}

// for benchmarking non-complex implementations
func benchNonComplexStore(createRuleStore ruleStoreCreator, b *testing.B) {
	// create rule store
	store := createRuleStore()

	// create rules
	rules := make([]Rule, benchRules)
	ruleType := ALLOW
	for idx := range rules {
		if idx >= benchRules / 2 {
			ruleType = BLOCK
		}
		rules[idx] = CreateRule(testutil.RandomDomain(), ruleType)
	}

	groups := []string{"alpha", "bravo", "charlie", "delta", "echo"}
	inGroups := []string{"zulu", "yankee", "november", "echo"}

	// load rules into store for each group
	for _, group := range groups {
		store.Load(group, rules)
	}

	// create list of queries
	queryData := make([]string, 100)
	for rdx := range queryData {
		queryData[rdx] = rules[(len(rules)/2)-(len(queryData)/2)+rdx].Text()
	}

	// start timer
	b.ResetTimer()
	b.ReportAllocs()

	// query
	for i := 0; i < b.N; i++ {
		store.IsMatchAny(inGroups, queryData[i%len(queryData)])
	}
}

func TestComplexRuleStore(t *testing.T) {

	ruleData := []ruleList{
		// whitelist checks are inverted but force a return without going through BLACK or BLOCK lists
		{group: "default", rule: "/^r.*\\..*/", ruleType: ALLOW, matching: []string{}, nomatches: []string{"ring.com", "rank.org", "riff.io"}},
		// black and blocklist checks are not
		{group: "default", rule: "/^r.*\\..*/", ruleType: BLOCK, matching: []string{"ring.com", "rank.org", "riff.io"}, nomatches: []string{"argument.com"}},
	}

	// with creator function
	testStore(ruleData, func() RuleStore { return CreateStore("") }, t)
}
