package engine

import (
	"net"
	"os"
	"testing"

	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/testutil"
	"github.com/chrisruffalo/gudgeon/util"
)

func parseIP(input string) *net.IP {
	ip := net.ParseIP(input)
	return &ip
}

func TestBasicEngine(t *testing.T) {
	config := testutil.TestConf(t, "testdata/simple.yml")
	defer os.RemoveAll(config.Home)

	// create engine from test config
	engine, err := NewEngine(config)
	if err != nil {
		t.Errorf("Could not create a new engine: %s", err)
		return
	}

	// test engine against block data (should not be blocked)
	if matched, _, _ := engine.IsDomainRuleMatched(parseIP("192.168.0.1"), "google.com"); matched == rule.MatchBlock {
		t.Errorf("Domain 'google.com' should not be blocked but it is")
	}
	if matched, _, _ := engine.IsDomainRuleMatched(parseIP("192.168.0.1"), "crittercism.com"); matched != rule.MatchBlock {
		t.Errorf("Domain 'crittercism.com' should be blocked but it is not")
	}

	engine.Shutdown()
}

func TestConsumerMatching(t *testing.T) {
	config := testutil.TestConf(t, "testdata/consumer_match.yml")
	defer os.RemoveAll(config.Home)

	// create engine from test config
	testEngine, err := NewEngine(config)
	if err != nil {
		t.Errorf("Could not create a new engine: %s", err)
	}

	// ip match data
	data := []struct {
		ip             string
		expectedGroups []string
	}{
		// ipv4
		{"192.168.0.1", []string{"alpha", "bravo"}},
		{"192.168.0.3", []string{"default"}},
		{"192.168.50.19", []string{"default"}},
		{"192.168.50.20", []string{"bravo", "charlie"}},
		{"192.168.50.25", []string{"bravo", "charlie"}},
		{"192.168.50.45", []string{"bravo", "charlie"}},
		{"192.168.50.90", []string{"bravo", "charlie"}},
		{"192.168.50.91", []string{"default"}},
		{"192.168.5.1", []string{"delta"}},
		{"192.168.5.2", []string{"delta"}},
		{"192.168.5.3", []string{"delta"}},
		{"192.168.5.128", []string{"delta"}},
		{"192.168.5.255", []string{"delta"}},
		// ipv6
		{"2001:0db8:0000:0000:0000:ff00:0042:8329", []string{"alpha6", "bravo6"}},
		{"2001:db8:0:0:0:ff00:42:8329", []string{"alpha6", "bravo6"}},
		{"2001:db8::ff00:42:8329", []string{"alpha6", "bravo6"}},
		{"2001:db8::ff00:42:8330", []string{"default"}},
		{"2001:0db8:0000:0000:0000:ff00:0090:0001", []string{"default"}},
		{"2001:0db8:0000:0000:0000:ff00:0090:0002", []string{"bravo6", "charlie6"}},
		{"2001:0db8:0000:0000:0000:ff00:0090:0003", []string{"bravo6", "charlie6"}},
		{"2001:0db8:0000:0000:0000:ff00:0091:0001", []string{"bravo6", "charlie6"}},
		{"2001:0db8:0000:0000:0000:ff00:0095:8329", []string{"bravo6", "charlie6"}},
		{"2001:0db8:0000:0000:0000:ff00:0099:0001", []string{"bravo6", "charlie6"}},
		{"2001:0db8:0000:0000:0000:ff00:009a:0001", []string{"default"}},
		{"2001:db8:0:0:0:ff00:aaaa:0", []string{"delta6"}},
		{"2001:db8:0:0:0:ff00:aaaa:ff0", []string{"delta6"}},
		{"2001:db8:0:0:0:ff00:aaaa:faa", []string{"delta6"}},
		{"2001:db8:0:0:0:ff00:aaaa:fff", []string{"delta6"}},
		// mixed support
		{"192.168.49.30", []string{"alpha", "alpha6"}},
		{"2002:0db8:0000:0000:0000:ff00:0090:0043", []string{"alpha", "alpha6"}},
	}

	// check data
	for _, value := range data {
		groupnames := testEngine.(*engine).getConsumerGroups(parseIP(value.ip))
		if len(groupnames) != len(value.expectedGroups) {
			t.Errorf("%s >> Expected values %s does not match %s {by length}", value.ip, value.expectedGroups, groupnames)
		} else {
			for _, eGroup := range value.expectedGroups {
				if !util.StringIn(eGroup, groupnames) {
					t.Errorf("%s >> Expected group %s in %s", value.ip, eGroup, groupnames)
				}
			}
		}
	}

	testEngine.Shutdown()
}

func TestReverseLookup(t *testing.T) {
	// create config and engine from config
	config := testutil.TestConf(t, "testdata/simple-reverse.yml")
	defer os.RemoveAll(config.Home)

	engine, err := NewEngine(config)
	if err != nil {
		t.Errorf("Could not create engine: %s", err)
	}

	data := []struct {
		ip       string
		expected string
	}{
		// all fqdns will have the last '.' appended to the last label, just be aware when adding more test cases
		{"127.0.0.1", "localhost."},
		{"172.0.0.1", "gudgeon.io."},
	}

	for _, d := range data {
		result := engine.Reverse(d.ip)

		if d.expected != result {
			t.Errorf("Did not get expected reverse lookup result for '%s': %s (got %s instead)", d.ip, d.expected, result)
		}
	}

	engine.Shutdown()
}
