package rule

import (
	"runtime"
	"testing"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/testutil"
)

const (
	// how many rules to benchmark with
	benchRules = 500000
)

type ruleList struct {
	group    string
	rule     string
	ruleType uint8
	blocked  []string
	allowed  []string
	nomatch  []string
}

type ruleStoreCreator func() RuleStore

func testStore(ruleData []ruleList, createRuleStore ruleStoreCreator, t *testing.T) {

	for _, data := range ruleData {
		// create rule and rule list
		ruleType := "block"
		if data.ruleType == ALLOW {
			ruleType = "allow"
		}

		rule := CreateRule(data.rule, data.ruleType)
		rules := []Rule{rule}

		// load rules into target store
		store := createRuleStore()
		lists := []*config.GudgeonList{&config.GudgeonList{Name: "Test List", Type: ruleType}}
		store.Load(nil, lists[0], rules)

		// check blocked
		for _, expectedBlock := range data.blocked {
			if MatchBlock != store.FindMatch(lists, expectedBlock) {
				t.Errorf("Rule '%s' of type %d expected to block '%s' but did not", data.rule, data.ruleType, expectedBlock)
			}
		}

		// check allowed
		for _, expectedAllow := range data.allowed {
			if MatchAllow != store.FindMatch(lists, expectedAllow) {
				t.Errorf("Rule '%s' of type %d expected to allow '%s' but did not", data.rule, data.ruleType, expectedAllow)
			}
		}

		// check no match ata ll
		for _, expectedNoMatch := range data.nomatch {
			if MatchNone != store.FindMatch(lists, expectedNoMatch) {
				t.Errorf("Rule '%s' of type %d expected to not match '%s' but did", data.rule, data.ruleType, expectedNoMatch)
			}
		}
	}
}

// for benchmarking non-complex implementations
func benchNonComplexStore(createRuleStore ruleStoreCreator, b *testing.B) {
	// create rule store
	store := createRuleStore()

	printMemUsage("before load", b)

	// create rules
	rules := make([]Rule, benchRules*2)
	ruleType := BLOCK
	for idx := 0; idx < benchRules*2; idx++ {
		if idx >= benchRules {
			ruleType = ALLOW
		}
		rules[idx] = CreateRule(testutil.RandomDomain(), ruleType)
	}

	lists := []*config.GudgeonList{
		&config.GudgeonList{Name: "Block", Type: "block"},
		&config.GudgeonList{Name: "Allow", Type: "allow"},
	}

	// load rules into store for each group
	store.Load(nil, lists[0], rules[:benchRules])
	store.Load(nil, lists[1], rules[benchRules:])

	// create list of queries
	queryData := make([]string, 100)
	for rdx := range queryData {
		queryData[rdx] = rules[(len(rules)/2)-(len(queryData)/2)+rdx].Text()
	}

	runtime.GC()
	printMemUsage("after load", b)

	// start timer
	b.ResetTimer()
	b.ReportAllocs()

	// query
	for i := 0; i < b.N; i++ {
		store.FindMatch(lists, queryData[i%len(queryData)])
	}

	// after test print memory usage too
	runtime.GC()
	printMemUsage("after test", b)
}

func TestComplexRuleStore(t *testing.T) {

	ruleData := []ruleList{
		// whitelist checks are inverted but force a return without going through BLACK or BLOCK lists
		{group: "default", rule: "/^r.*\\..*/", ruleType: ALLOW, blocked: []string{}, allowed: []string{"ring.com", "rank.org", "riff.io"}, nomatch: []string{}},
		// black and blocklist checks are not
		{group: "default", rule: "/^r.*\\..*/", ruleType: BLOCK, blocked: []string{"ring.com", "rank.org", "riff.io"}, allowed: []string{}, nomatch: []string{"argument.com"}},
	}

	// with creator function
	testStore(ruleData, func() RuleStore { return CreateStore("") }, t)
}

func printMemUsage(msg string, b *testing.B) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	b.Logf("%s: Alloc = %v MiB", msg, bToMb(m.Alloc))
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
