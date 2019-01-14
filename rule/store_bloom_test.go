package rule

import (
	"testing"
)

func TestBloomRuleStore(t *testing.T) {

	ruleData := []ruleList{
		// whitelist checks are inverted but force a return without going through BLACK or BLOCK lists
		{group: "default", rule: "rate.com", ruleType: ALLOW, blocked: []string{}, allowed: []string{"we.rate.com", "no.rate.com", "rate.com"}, nomatch: []string{"crate.com", "rated.com"}},
		// black and blocklist checks are not
		{group: "default", rule: "bonkers.com", ruleType: BLOCK, blocked: []string{"text.bonkers.com", "bonkers.com"}, nomatch: []string{"argument.com", "boop.com", "krunch.io", "bonk.com", "bonkerss.com"}},
	}

	testStore(ruleData, func() RuleStore { return CreateStore("bloom") }, t)
}

func BenchmarkBloomRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return CreateStore("bloom") }, b)
}
