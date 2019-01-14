package rule

import (
    "os"
	"testing"

    "github.com/chrisruffalo/gudgeon/downloader"
    "github.com/chrisruffalo/gudgeon/testutil"
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

func TestIsInListFile(t *testing.T) {
    config := testutil.Conf(t, "testdata/listinfile.yml")
    defer os.RemoveAll(config.Home)
    list := config.Lists[0]
    downloader.Download(config, list)

    data := []struct{
        domain string
        expected bool
    }{
        {"z.zeroredirect.com", true},
        {"google.com", false},
        {"zeroredirect", false},
    }

    for _, d := range data {
        result := isInListFile(d.domain, config, list)
        if result != d.expected {
            t.Errorf("error with %s search expected:%v but got:%v", d.domain, d.expected, result)
        }
    }
}
