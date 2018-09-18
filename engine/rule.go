package engine

import (
	"strings"
)

// converts a rule, which might be a host name rule or a comment or something, into a real rule that can be used
func ParseRule(rule string) string {
	// a rule that starts with a comment sign is parsed as an empty string which should be ignored by other parts of the API
	if strings.HasPrefix(rule, "#") || strings.HasPrefix(rule, "//") {
		return ""
	}

	// a rule that can be split on spaces is more complicated, for right now just take everything after the first space
	split := strings.Split(rule, " ")
	if len(split) > 1 {
		rule = strings.Join(split[1:], " ")
	}

	return strings.TrimSpace(rule)
}

func IsComplexRule(rule string) bool {
	return false
}

func matchComplexRule(value string, rule string) bool {
	return false
}

func IsMatch(value string, rule string) bool {
	if IsComplexRule(rule) {
		return matchComplexRule(value, rule)
	}
	// check to see if the value matches the rule OR if the 
	// value has a suffix that matches the "." + rule so that
	// "google.com" blocks "subdomain.google.com" and "google.com"
	return value == rule || strings.HasSuffix(value, "." + rule)
}