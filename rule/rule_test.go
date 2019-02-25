package rule

import (
	"testing"
)

func TestParseLine(t *testing.T) {
	data := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"#blah blah", ""},
		{"thing #blah blah", "thing"},
		{"     thing         #blah blah", "thing"},
		{"//result", ""},
		{"stuff result", "result"},
		{"stuff rule //comment", "rule"},
		{"127.0.0.1 gone.ads.io //comment", "gone.ads.io"},
		{"here.ads.io //comment", "here.ads.io"},
		{"google.com", "google.com"},
		{"                                    google.com    ", "google.com"},
		{"         #                          google.com    ", ""},
		{"  //     #                          google.com    ", ""},
	}

	for _, d := range data {
		result := ParseLine(d.input)
		if d.expected != result {
			t.Errorf("Input '%s' should have '%s' but got '%s'", d.input, d.expected, result)
		}
	}
}
