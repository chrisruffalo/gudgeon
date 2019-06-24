package util

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	Hours = 1
	Day   = 24
	Week  = 24 * 7
)

// used for duration parsing for h, d, w instead of
// compiling all of them on the fly
var regexpTokens = []*regexp.Regexp{
	regexp.MustCompile("([0-9]+?)h"),
	regexp.MustCompile("([0-9]+?)d"),
	regexp.MustCompile("([0-9]+?)w"),
}

func ParseDuration(input string) (time.Duration, error) {
	durations := []int{
		Hours,
		Day,
		Week,
	}

	// hours count
	hours := 0

	for idx, tokenRegexp := range regexpTokens {
		tokenMatches := tokenRegexp.FindStringSubmatch(input)
		if len(tokenMatches) > 0 {
			value, err := strconv.Atoi(tokenMatches[1])
			if err == nil {
				hours += value * durations[idx]
				input = strings.Replace(input, tokenMatches[0], "", 1)
			}
		}
	}

	// prepend hours
	if hours > 0 {
		input = fmt.Sprintf("%dh%s", hours, input)
	}

	return time.ParseDuration(input)
}
