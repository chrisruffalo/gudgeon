package benchmarks

import (
	"io/ioutil"
	"sort"
	"strings"

	"github.com/spaolacci/murmur3"
)

type keephash struct {
	file []uint64
}

func (keephash *keephash) Id() string {
	return "keephash"
}

func hash(input string) uint64 {
	return murmur3.Sum64([]byte(input))
}

func (keephash *keephash) Load(inputfile string) error {
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}
	array := strings.Split(string(content), "\r")
	output := make([]uint64, len(array))
	for idx, item := range array {
		output[idx] = hash(strings.TrimSpace(item))
	}
	sort.Slice(output, func(i, j int) bool { return output[i] < output[j] })

	keephash.file = output

	return nil
}

func found(array []uint64, item uint64) bool {
	return array[sort.Search(len(array), func(i int) bool { return array[i] >= item })] == item
}

func (keephash *keephash) Test(forMatch string) (bool, error) {
	return found(keephash.file, hash(forMatch)) || found(keephash.file, hash(rootdomain(forMatch))), nil
}

func (keephash *keephash) Teardown() error {
	return nil
}