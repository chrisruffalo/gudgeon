package rule

import (
	"testing"
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
