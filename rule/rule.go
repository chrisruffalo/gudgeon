package rule

import (
	"regexp"
	"strings"

	"github.com/ryanuber/go-glob"
)

const (
	wildcard = "*"
	comment = "#"
    altComment = "//"
	regex = "/"
)

type Rule interface {
	IsMatch(sample string) bool
	IsComplex() bool
}

type baseRule struct {
	text string
}

type textMatchRule struct {
	baseRule
}

type wildcardMatchRule struct {
	baseRule
}

type regexMatchRule struct {
	baseRule
	regexp *regexp.Regexp
}

func Create(rule string) Rule {
	// a rule that starts with a comment sign is parsed as an empty string which should be ignored by other parts of the API
	if strings.HasPrefix(rule, comment) || strings.HasPrefix(rule, altComment) {
		return nil
	}

	// regex rules start and end with "/" to denote them that way
	if strings.HasPrefix(rule, regex) && strings.HasSuffix(rule, regex) {
		return createRegexMatchRule(rule)
	}

	// wildcard rules have wildcards in them (only * is supported)
	if strings.Contains(rule, wildcard) {
		return createWildcardMatchRule(rule)
	}

	// all other rules are straight text match
	return createTextMatchRule(rule)
}

// =================================================================
// Rule Creation
// =================================================================
func createTextMatchRule(rule string) Rule {
	newRule := new(textMatchRule)
	newRule.text = rule
	return newRule
}

func createWildcardMatchRule(rule string) Rule {
	newRule := new(wildcardMatchRule)
	newRule.text = rule
	return newRule
}

func createRegexMatchRule(rule string) Rule {
	newRule := new(regexMatchRule)
	newRule.text = rule
	cRegex, err := regexp.Compile(rule[1:len(rule)-1])
	newRule.regexp = cRegex
	if err != nil {
		return nil
	}
	return newRule
}

// =================================================================
// Rule Complexity
// =================================================================
func (rule *textMatchRule) IsComplex() bool {
	return false
}

func (rule *wildcardMatchRule) IsComplex() bool {
	return true
}

func (rule *regexMatchRule) IsComplex() bool {
	return true
}

// =================================================================
// Rule Matching
// =================================================================
func (rule *textMatchRule) IsMatch(sample string) bool {
	// check to see if the value matches the rule OR if the 
	// value has a suffix that matches the "." + rule so that
	// "google.com" blocks "subdomain.google.com" and "google.com"
	return rule.text == sample || strings.HasSuffix(sample, "." + rule.text)
}

func (rule *wildcardMatchRule) IsMatch(sample string) bool {
	return glob.Glob(rule.text, sample)
}

func (rule *regexMatchRule) IsMatch(sample string) bool {
	return rule.regexp.MatchString(sample)
}
