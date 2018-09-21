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
	return sort.Search(len(array), func(i int) bool { return array[i] >= item }) >= 0
}

func (keephash *keephash) Test(forMatch string) (bool, error) {
	rootdomain := hash(rootdomain(forMatch))
	matchhash := hash(forMatch)
	return found(keephash.file, matchhash) || found(keephash.file, rootdomain), nil
}