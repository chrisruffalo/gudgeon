package util

import (
	"testing"
	"time"
)

func TestParseDuration(t *testing.T) {
	data := []struct {
		input    string
		expected time.Duration
	}{
		{"26h", 26 * time.Hour},
		{"5d3h", 24*5*time.Hour + 3*time.Hour},
		{"1w", 24 * 7 * time.Hour},
		{"1w2d", 24 * 9 * time.Hour},
		{"3w10d", 24 * 31 * time.Hour},
		{"12w7d12h4m22s", 24*7*12*time.Hour + 24*7*time.Hour + 12*time.Hour + 4*time.Minute + 22*time.Second},
	}

	for _, d := range data {
		output, err := ParseDuration(d.input)
		if err != nil {
			t.Errorf("Error parsing duration: %s", err)
		} else if d.expected != output {
			t.Errorf("Expected %v from input '%s' but got %v", d.expected, d.input, output)
		}
	}
}
