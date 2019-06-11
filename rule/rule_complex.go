package rule

import (
	"regexp"
	"strings"

	"github.com/ryanuber/go-glob"
)

type ComplexRule interface {
	IsMatch(sample string) bool
	Text() string
}

type complexRule struct {
	text string
}

type wildcardMatchRule struct {
	complexRule
}

type regexMatchRule struct {
	complexRule
	regexp *regexp.Regexp
}

func createComplexRule(rule string) ComplexRule {
	if strings.HasPrefix(rule, ruleRegex) && strings.HasSuffix(rule, ruleRegex) {
		// regex rules start and end with "/" to denote them that way
		return createRegexMatchRule(rule)
	} else if strings.Contains(rule, ruleGlob) {
		// wildcard rules have wildcards in them (only * is supported)
		return createWildcardMatchRule(rule)
	}

	// return rule
	return nil
}

// =================================================================
// Rule Creation
// =================================================================
func createWildcardMatchRule(rule string) ComplexRule {
	newRule := new(wildcardMatchRule)
	newRule.text = rule
	return newRule
}

func specifyRegexOnlyRule(rule string) ComplexRule {
	newRule := &regexMatchRule{}
	newRule.text = rule
	cRegex, err := regexp.Compile(rule)
	if err != nil {
		return nil
	}
	newRule.regexp = cRegex
	return newRule
}

func createRegexMatchRule(rule string) ComplexRule {
	return specifyRegexOnlyRule(rule[1 : len(rule)-1])
}

// =================================================================
// Base operations for Rule identification (mainly for backing stores)
// =================================================================
func (rule *complexRule) Text() string {
	return rule.text
}

// =================================================================
// Rule Matching
// =================================================================
func (rule *wildcardMatchRule) IsMatch(sample string) bool {
	return glob.Glob(rule.text, sample)
}

func (rule *regexMatchRule) IsMatch(sample string) bool {
	return rule.regexp.MatchString(sample)
}
