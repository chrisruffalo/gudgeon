package rule

import (
	"testing"
)

func TestRuleCreation(t *testing.T) {
	rule := CreateRule("simple.domain.com", BLOCKLIST)
	if rule == nil || rule.IsComplex() {
		t.Errorf("Simple rule should not be a complex rule")
	}

	rule = CreateRule("/.*google.*/", BLOCKLIST) 
	if rule == nil || !rule.IsComplex() {
		t.Errorf("Regex rule should not be nil and should be complex")
	}

	rule = CreateRule("*.google.com", BLOCKLIST)
	if rule == nil || !rule.IsComplex(){
		t.Errorf("Wildcard rule should not be nil and should be complex")
	}
}

type domainData struct {
	domain string
	expected bool
}

func testRuleMatching(testType string, text string, data []domainData, t *testing.T) {
	rule := CreateRule(text, BLOCKLIST)
	for _, d := range data {
		result := rule.IsMatch(d.domain)
		if result != d.expected {
			t.Errorf("%s - (rule: %s) - IsMatch(%s) was %t but expected %t", testType, text, d.domain, result, d.expected)
		}
	}
}

func TestTextRuleMatching(t *testing.T) {
	data := []domainData {
		{ domain: "google.com", expected: true },
		{ domain: "yahoo.com", expected: false },
		{ domain: "mail.google.com", expected: true },
	}
	testRuleMatching("simple", "google.com", data, t)
}

func TestWildcardRuleMatching(t *testing.T) {
	data := []domainData {
		{ domain: "google.com", expected: false },
		{ domain: "ads.google.com", expected: true },
		{ domain: "ads.yahoo.com", expected: true },
		{ domain: "ads.yahoo.org", expected: false },
		{ domain: "ads.com", expected: false },
	}
	testRuleMatching("wildcard", "a*.*.com", data, t)
}

func TestRegexRuleMatching(t *testing.T) {
	data := []domainData {
		{ domain: "ripple.com", expected: true },
		{ domain: "rack.com", expected: true },
		{ domain: "frack.com", expected: false },
		{ domain: "rrrrr.com.co", expected: false },
	}
	testRuleMatching("regex", "/^r.*\\.com$/", data, t)
}