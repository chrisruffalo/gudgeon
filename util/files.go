package util

import (
	"io/ioutil"
	"os"
	"path"
	"strings"
)

func GetFileAsArray(inputfile string) ([]string, error) {
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return []string{}, err
	}
	return strings.Split(string(content), "\n"), nil
}

// clears the contents of a directory but leaves it
func ClearDirectory(inputdir string) {
	dir, err := ioutil.ReadDir(inputdir)
	if err != nil {
		return
	}
	for _, d := range dir {
		os.RemoveAll(path.Join([]string{inputdir, d.Name()}...))
	}
}
