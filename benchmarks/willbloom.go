package benchmarks

import (
	"bufio"
	"os"

	"github.com/willf/bloom"
)

type willbloom struct {
	filter *bloom.BloomFilter
}

func (willbloom *willbloom) Load(inputfile string) error {
	data, err := os.Open(inputfile)
	if err != nil {
		return err
	}
	
	// check number of lines
	totalLines := uint64(0)
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		scanner.Text()
		totalLines++
	}
	data.Close()

	// build bloom
	filter := bloom.NewWithEstimates(uint(totalLines), 0.00001)

	data, err = os.Open(inputfile)
	if err != nil {
		return err
	}
	defer data.Close()

	scanner = bufio.NewScanner(data)
	for scanner.Scan() {
		filter.AddString(scanner.Text())
	}

	// save filter
	willbloom.filter = filter

	return nil
}

func (willbloom *willbloom) Test(forMatch string) (bool, error) {
	// either matches the domain or the root domain is in the match
	return willbloom.filter.TestString(forMatch) || willbloom.filter.TestString(rootdomain(forMatch)), nil
}