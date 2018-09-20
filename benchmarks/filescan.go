package benchmarks

import (
	"bufio"
	"os"
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

	rootdomain := rootdomain(forMatch)

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		text := scanner.Text()
		if text == forMatch || text == rootdomain {
			return true, nil
		}
	}
	return false, nil
}