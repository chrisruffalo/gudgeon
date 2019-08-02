package util

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
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
		_ = os.RemoveAll(path.Join([]string{inputdir, d.Name()}...))
	}
}

const _lineBufferByteSize = 32 * 1024

var _lineSep = []byte{'\n'}

var lineCountPool = sync.Pool{New: func() interface{} {
	return make([]byte, _lineBufferByteSize)
}}

// count file lines
// from: https://stackoverflow.com/a/24563853
func LineCount(inputfile string) (uint, error) {
	r, err := os.Open(inputfile)
	if err != nil {
		return 0, err
	}
	defer r.Close()

	buf := lineCountPool.Get().([]byte)
	defer lineCountPool.Put(buf)

	count := uint(0)

	for {
		c, err := r.Read(buf)
		count += uint(bytes.Count(buf[:c], _lineSep))

		switch {
		case err == io.EOF:
			return count, nil

		case err != nil:
			return count, err
		}
	}
}
