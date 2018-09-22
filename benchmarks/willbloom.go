package benchmarks

import (
	"bufio"
	"os"

	"github.com/willf/bloom"
)

type willbloom struct {
	rate float64
	filter *bloom.BloomFilter
}

func (willbloom *willbloom) Id() string {
	return "willbloom"
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

	// build bloom with non-zero rate
	if willbloom.rate == 0 {
		willbloom.rate = 0.01 // 1%
	}
	filter := bloom.NewWithEstimates(uint(totalLines), willbloom.rate)

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

func (willbloom *willbloom) Teardown() error {
	return nil
}