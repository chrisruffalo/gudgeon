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

func RootDomain(domain string) string {
	split := strings.Split(domain, ".") 
	if len(split) >= 2 {
		return strings.Join(split[len(split)-2:], ".")
	}
	return domain
}