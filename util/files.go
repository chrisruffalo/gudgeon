package util

import (
	"io/ioutil"
	"strings"
)

func GetFileAsArray(inputfile string) ([]string, error) {
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(string(content), "\n"), nil
}
