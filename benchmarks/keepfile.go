package benchmarks

import (
	"io/ioutil"
	"sort"
	"strings"
)

type keepfile struct {
	file []string
}

func (keepfile *keepfile) Id() string {
	return "keepfile"
}

func (keepfile *keepfile) Load(inputfile string, testdir string) error {
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}
	array := strings.Split(string(content), "\r")
	for idx, item := range array {
		array[idx] = strings.TrimSpace(item)
	}
	sort.Strings(array)

	keepfile.file = array

	return nil
}

func (keepfile *keepfile) Test(forMatch string) (bool, error) {
	rootdomain := rootdomain(forMatch)
	return keepfile.file[sort.SearchStrings(keepfile.file, forMatch)] == forMatch || keepfile.file[sort.SearchStrings(keepfile.file, rootdomain)] == rootdomain, nil
}

func (keepfile *keepfile) Teardown() error {
	return nil
}