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

func ParseDuration(input string) (time.Duration, error) {
	// these are what are going to be converted
	tokens := []string{"h", "d", "w"}
	durations := []int{
		Hours,
		Day,
		Week,
	}

	// hours count
	hours := 0

	for idx, token := range tokens {
		regexp := regexp.MustCompile("([0-9]+?)" + token)
		tokenMatches := regexp.FindStringSubmatch(input)
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
