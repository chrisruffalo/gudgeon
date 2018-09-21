package benchmarks

import (
	"io/ioutil"
	"strings"
)

type keepfileslow struct {
	file []string
}

func (keepfile *keepfileslow) Load(inputfile string) error {
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}
	array := strings.Split(string(content), "\r")
	for idx, item := range array {
		array[idx] = strings.TrimSpace(item)
	}
	keepfile.file = array

	return nil
}

func (keepfile *keepfileslow) Test(forMatch string) (bool, error) {
	rootdomain := rootdomain(forMatch)
	for _, item := range keepfile.file {
		if item == forMatch || item == rootdomain {
			return true, nil
		}
	}
	return false, nil
}