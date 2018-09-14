package engine

import (
	"strings"
)

// converts a rule, which might be a host name rule or a comment or something, into a real rule that can be used
func ParseRule(rule string) string {
	// a rule that starts with a comment sign is parsed as an empty string which is ignored
	if strings.HasPrefix(rule, "#") {
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

func MatchesRule(value string, rule string) bool {
	return value == rule
}