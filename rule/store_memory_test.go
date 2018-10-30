package rule

import (
	"testing"
)

func TestMemoryRuleStore(t *testing.T) {

	ruleData := []ruleList {
		// whitelist checks are inverted but force a return without going through BLACK or BLOCK lists
		{ group: "default", rule: "rate.com", ruleType: WHITELIST, matching: []string{}, nomatches: []string{"we.rate.com", "no.rate.com", "rate.com"} },
		// black and blocklist checks are not
		{ group: "default", rule: "bonkers.com", ruleType: BLACKLIST, matching: []string{"bonkers.com"}, nomatches: []string{"argument.com", "boop.com", "krunch.io"} },
	}

	testStore(ruleData, func() RuleStore { return new(memoryStore) }, t)
}