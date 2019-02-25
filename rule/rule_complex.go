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

func CreateComplexRule(rule string) ComplexRule {
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

func createRegexMatchRule(rule string) ComplexRule {
	newRule := new(regexMatchRule)
	newRule.text = rule
	cRegex, err := regexp.Compile(rule[1 : len(rule)-1])
	newRule.regexp = cRegex
	if err != nil {
		return nil
	}
	return newRule
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
