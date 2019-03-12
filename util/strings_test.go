package util

import (
    "testing"
)

func TestTrimComment(t *testing.T) {
    data := []struct {
        input    string
        expected string
        prefixes []string
    } {
        { "// blah blah", "", []string{} },
        { "blah //", "blah ", []string{} },
        { "zi//p", "zi", []string{} },
        { "# adlkjfasfd", "", []string{} },
        {" thing  # comment", " thing  ", []string{} },
        {"; nobind", "; nobind", []string{} },
        {"; nobind", "", []string{";"} },
        {"  a test  ; nobind", "  a test  ", []string{";"} },
    }

    for _, d := range data {
        actual := TrimComments(d.input, d.prefixes...)
        if d.expected != actual {
            t.Errorf("Failed to trim comment '%s', expected '%s' but got '%s'", d.input, d.expected, actual)
        }
    }
}