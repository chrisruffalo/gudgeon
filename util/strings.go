package util

import (
	"strings"
)

// determines if the given string is in the array of strings
func StringIn(value string, in []string) bool {
	for _, test := range in {
		if value == test {
			return true
		}
	}
	return false
}

func StartsWithAny(value string, any []string) bool {
	for _, test := range any {
		if strings.HasPrefix(value, test) {
			return true
		}
	}
	return false
}

// adapted from https://groups.google.com/d/msg/golang-nuts/oPuBaYJ17t4/PCmhdAyrNVkJ
func ReverseString(input string) string {
	// Get Unicode code points.
	n := 0
	rune := make([]rune, len(input))
	for _, r := range input {
		rune[n] = r
		n++
	}
	rune = rune[0:n]

	// Reverse
	for i := 0; i < n/2; i++ {
		rune[i], rune[n-1-i] = rune[n-1-i], rune[i]
	}
	// Convert back to UTF-8.
	output := string(rune)

	return output
}

// takes a line and removes commented section
var DefaultCommentPrefixes = []string {
	"#",
	"//",
}
func TrimComments(line string, commentPrefixes ...string) string {
	if len(commentPrefixes) < 1 {
		commentPrefixes = DefaultCommentPrefixes
	}

	// if it starts with any of the comment prefixes, return an empty line
	if StartsWithAny(line, commentPrefixes) {
		return ""
	}

	// otherwise if the comment prefix is in the line chop everything before it
	for _, prefix := range commentPrefixes {
		if strings.Contains(line, prefix) {
			line = line[0:strings.Index(line,prefix)]
		}
	}

	return line
}