package util

import (
	"github.com/mina86/unsafeConvert"
	"strings"
	"unsafe"
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

// takes a line and removes commented section
var DefaultCommentPrefixes = []string{
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
			line = line[0:strings.Index(line, prefix)]
		}
	}

	return line
}

var emptyByte = make([]byte, 0)

func SaferBytes(input string) []byte {
	if "" == input {
		return emptyByte
	}
	return unsafeConvert.Bytes(input)
}

// from: https://github.com/golang/go/issues/25484
func ByteSliceToString(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}
