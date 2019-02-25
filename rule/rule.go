package rule

import (
	"strings"
)

var commentChars = []string{"#", "//"}

const (
	// the constant that means ALLOW after pasring "allow" or "block"
	ALLOW = uint8(1)
	// the constant that means BLOCK after pasring "allow" or "block"
	BLOCK = uint8(0)
	// the string that represents "allow", all other results are treated as "block"
	ALLOWSTRING = "allow"

	ruleRegex = "/"
	ruleGlob  = "*"
)

func ParseType(listType string) uint8 {
	if strings.EqualFold(ALLOWSTRING, listType) {
		return ALLOW
	}
	return BLOCK
}

// parse a line, as from a file, and return the part that represents the rule
func ParseLine(line string) string {
	line = strings.TrimSpace(line)

	// remove everything after any comment on the line
	for _, char := range commentChars {
		if cIndex := strings.Index(line, char); cIndex >= 0 {
			line = strings.TrimSpace(line[0:cIndex])
		}
	}

	// a rule that can be split on spaces is more complicated, for right now just take everything after the first space
	// this is because of the common rule formats are either
	// <some ip> <domain name>
	// or
	// <domain name>
	// and we are interpreting these as rules
	split := strings.Split(line, " ")
	if len(split) > 1 {
		line = strings.Join(split[1:], " ")
	}

	// return rule
	return line
}

func IsComplex(ruleText string) bool {
	return strings.Contains(ruleText, ruleGlob) || (strings.HasPrefix(ruleText, ruleRegex) && strings.HasSuffix(ruleText, ruleRegex))
}
