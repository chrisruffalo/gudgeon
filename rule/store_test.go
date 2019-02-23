package rule

import (
	"fmt"
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
	rules    []string
	ruleType uint8
	blocked  []string
	allowed  []string
	nomatch  []string
}

// quick/dirty rule tests that all stores should pass
var defaultRuleData = []ruleList {
	// allowlist checks are inverted
	{group: "default", rules: []string{"rate.com"}, ruleType: ALLOW, blocked: []string{}, allowed: []string{"we.rate.com", "no.rate.com", "rate.com"}, nomatch: []string{"crate.com", "rated.com"}},
	// blocklist checks are not
    {group: "default", rules: []string{"rate.com","gorp.com"}, ruleType: BLOCK, blocked: []string{"we.rate.com", "no.rate.com", "rate.com", "gorp.com", "clog.gorp.com"}, allowed: []string{}, nomatch: []string{"crate.com", "rated.com", "orp.com"}},
	{group: "default", rules: []string{"bonkers.com"}, ruleType: BLOCK, blocked: []string{"text.bonkers.com", "bonkers.com"}, nomatch: []string{"argument.com", "boop.com", "krunch.io", "bonk.com", "bonkerss.com"}},
}


type ruleStoreCreator func() RuleStore

func testStore(ruleData []ruleList, createRuleStore ruleStoreCreator, t *testing.T) {
	tmpDir := testutil.TempDir()

	// create single store for test
	store := createRuleStore()

	for idx, data := range ruleData {
		// create rule and rule list
		ruleType := "block"
		if data.ruleType == ALLOW {
			ruleType = "allow"
		}

		rules := make([]Rule, len(data.rules))
		for idx, ruleText := range data.rules {
			rules[idx] = CreateRule(ruleText, data.ruleType)
		}

		lists := []*config.GudgeonList{&config.GudgeonList{Name: fmt.Sprintf("Test List %d", idx), Type: ruleType}}
		// load rules into target store
		store.Load(nil, lists[0], tmpDir, rules)

		// check blocked
		for _, expectedBlock := range data.blocked {
			result, _, _ := store.FindMatch(lists, expectedBlock)
			if MatchBlock != result {
				t.Errorf("Rules of type %d expected to block '%s' but did not", data.ruleType, expectedBlock)
			}
		}

		// check allowed
		for _, expectedAllow := range data.allowed {
			result, _, _ := store.FindMatch(lists, expectedAllow)
			if MatchAllow != result {
				t.Errorf("Rules of type %d expected to allow '%s' but did not", data.ruleType, expectedAllow)
			}
		}

		// check no match ata ll
		for _, expectedNoMatch := range data.nomatch {
			result, _, _ := store.FindMatch(lists, expectedNoMatch)
			if MatchNone != result {
				t.Errorf("Rules of type %d expected to not match '%s' but did", data.ruleType, expectedNoMatch)
			}
		}
	}
}

// for benchmarking non-complex implementations
func benchNonComplexStore(createRuleStore ruleStoreCreator, b *testing.B) {
	tmpDir := testutil.TempDir()

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
	store.Load(nil, lists[0], tmpDir, rules[:benchRules])
	store.Load(nil, lists[1], tmpDir, rules[benchRules:])

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
		{group: "default", rules: []string{"/^r.*\\..*/"}, ruleType: ALLOW, blocked: []string{}, allowed: []string{"ring.com", "rank.org", "riff.io"}, nomatch: []string{}},
		// black and blocklist checks are not
		{group: "default", rules: []string{"/^r.*\\..*/"}, ruleType: BLOCK, blocked: []string{"ring.com", "rank.org", "riff.io"}, allowed: []string{}, nomatch: []string{"argument.com"}},
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
