package benchmarks

import (
	"bufio"
	"os"

	"github.com/chrisruffalo/gudgeon/engine"
)

type fileScan struct {
	filename string
}

func (fileScan *fileScan) Load(inputfile string) error {
	fileScan.filename = inputfile
	return nil
}

func (fileScan *fileScan) Test(forMatch string) (bool, error) {
	data, err := os.Open(fileScan.filename)
	if err != nil {
		return false, err
	}
	defer data.Close()

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		if engine.IsMatch(forMatch, scanner.Text()) {
			return true, nil
		}
	}
	return false, nil
}