package rule

import (
	"strings"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	ruleRegex = "/"
	ruleGlob  = "*"
)

// parse a line, as from a file, and return the part that represents the rule
func ParseLine(line string) string {
	// remove everything after any comment on the line
	line = util.TrimComments(line)

	// trim spaces
	line = strings.TrimSpace(line)

	// a rule that can be split on spaces is more complicated, for right now just take everything after the first space
	// this is because of the common rule formats are either
	// <some ip> <domain name>
	// or
	// <domain name>
	// and we are interpreting these as rules
	if idx := strings.Index(line, " "); idx > -1 {
		line = line[idx+1:]
	}

	// return rule
	return line
}

func IsComplex(ruleText string) bool {
	return strings.Contains(ruleText, ruleGlob) || (strings.HasPrefix(ruleText, ruleRegex) && strings.HasSuffix(ruleText, ruleRegex))
}
