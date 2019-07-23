package rule

import (
	"fmt"
	"os"
	"runtime"
	"testing"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/testutil"
)

const (
	// how many rules to benchmark with
	benchRules = 750000
)

type ruleList struct {
	group    string
	rules    []string
	ruleType config.ListType
	blocked  []string
	allowed  []string
	nomatch  []string
}

// quick/dirty rule tests that all stores should pass
var defaultRuleData = []ruleList{
	// allowlist checks are inverted
	{group: "default", rules: []string{"rate.com"}, ruleType: config.ALLOW, blocked: []string{}, allowed: []string{"we.rate.com", "no.rate.com", "rate.com"}, nomatch: []string{"crate.com", "rated.com"}},
	// blocklist checks are not
	{group: "default", rules: []string{"rate.com", "gorp.com"}, ruleType: config.BLOCK, blocked: []string{"we.rate.com", "no.rate.com", "rate.com", "gorp.com", "clog.gorp.com"}, allowed: []string{}, nomatch: []string{"crate.com", "rated.com", "orp.com"}},
	{group: "default", rules: []string{"bonkers.com"}, ruleType: config.BLOCK, blocked: []string{"text.bonkers.com", "bonkers.com"}, nomatch: []string{"argument.com", "boop.com", "krunch.io", "bonk.com", "bonkerss.com"}},
}

type ruleStoreCreator func() Store

func testStore(ruleData []ruleList, createRuleStore ruleStoreCreator, t *testing.T) {
	tmpDir := testutil.TempDir()
	defer os.RemoveAll(tmpDir)

	for idx, data := range ruleData {
		// create rule and rule list
		ruleType := config.BLOCKSTRING
		if data.ruleType == config.ALLOW {
			ruleType = config.ALLOWSTRING
		}

		// create single store for test
		store := createRuleStore()

		// create and verify/update lists
		lists := []*config.GudgeonList{
			{Name: fmt.Sprintf("Test List %d", idx), Type: string(ruleType)},
		}
		for _, list := range lists {
			list.VerifyAndInit()
		}

		// load rules into target store
		store.Init(tmpDir, nil, lists)
		for _, rule := range data.rules {
			store.Load(lists[0], rule)
		}
		store.Finalize(tmpDir, lists)

		// check blocked
		for _, expectedBlock := range data.blocked {
			result, _, _ := store.FindMatch(lists, expectedBlock)
			if MatchBlock != result {
				t.Errorf("Rules of type %d in list %s expected to block '%s' but did not", data.ruleType, lists[0].CanonicalName(), expectedBlock)
			}
		}

		// check allowed
		for _, expectedAllow := range data.allowed {
			result, _, _ := store.FindMatch(lists, expectedAllow)
			if MatchAllow != result {
				t.Errorf("Rules of type %d in list %s expected to allow '%s' but did not", data.ruleType, lists[0].CanonicalName(), expectedAllow)
			}
		}

		// check no match ata ll
		for _, expectedNoMatch := range data.nomatch {
			result, _, _ := store.FindMatch(lists, expectedNoMatch)
			if MatchNone != result {
				t.Errorf("Rules of type %d in list %s expected to not match '%s' but did", data.ruleType, lists[0].CanonicalName(), expectedNoMatch)
			}
		}

		store.Close()
	}
}

// for benchmarking non-complex implementations
func benchNonComplexStore(createRuleStore ruleStoreCreator, b *testing.B) {
	tmpDir := testutil.TempDir()
	defer os.RemoveAll(tmpDir)

	// create rule store
	store := createRuleStore()

	printMemUsage("before load", b)

	lists := []*config.GudgeonList{
		{Name: "Block1", Type: "block"},
		{Name: "Block2", Type: "block"},
		{Name: "Block3", Type: "block"},
		{Name: "Allow1", Type: "allow"},
		{Name: "Allow2", Type: "allow"},
		{Name: "Allow3", Type: "allow"},
	}
	for _, list := range lists {
		list.VerifyAndInit()
	}

	store.Init(tmpDir, nil, lists)

	// create rules
	queryData := make([]string, 100)
	for idx := 0; idx < benchRules; idx++ {
		testDomain := testutil.RandomDomain()
		store.Load(lists[idx%6], testDomain)
		if idx < len(queryData) {
			queryData[idx] = testDomain
		}
	}

	store.Finalize(tmpDir, lists)

	runtime.GC()
	printMemUsage("after load", b)

	// start timer
	b.ResetTimer()
	b.ReportAllocs()

	// benchmark query
	for i := 0; i < b.N; i++ {
		store.FindMatch(lists, queryData[i%len(queryData)])
	}

	// after test print memory usage too
	runtime.GC()
	printMemUsage("after test", b)
}

func TestComplexRuleStore(t *testing.T) {

	ruleData := []ruleList{
		// whitelist checks are inverted but force a return without going through BLACK or config.BLOCK lists
		{group: "default", rules: []string{"/^r.*\\..*/"}, ruleType: config.ALLOW, blocked: []string{}, allowed: []string{"ring.com", "rank.org", "riff.io"}, nomatch: []string{}},
		// black and blocklist checks are not
		{group: "default", rules: []string{"/^r.*\\..*/"}, ruleType: config.BLOCK, blocked: []string{"ring.com", "rank.org", "riff.io"}, allowed: []string{}, nomatch: []string{"argument.com"}},
	}

	// with creator function
	testStore(ruleData, func() Store { return &complexStore{} }, t)
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
