package benchmarks

import (
	"bufio"
	"os"

	"github.com/alecthomas/mph"
)

var empty = []byte{}

type minhash struct {
	chd *mph.CHD
}

func (minhash *minhash) Load(inputfile string) error {
	data, err := os.Open(inputfile)
	if err != nil {
		return err
	}
	defer data.Close()

	// build mph
	b := mph.Builder()
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		b.Add([]byte(scanner.Text()), empty)
	}
	minhash.chd, err = b.Build()
	if err != nil {
		return err
	}

	return nil
}

func (minhash *minhash) Test(forMatch string) (bool, error) {
	// either matches the domain or the root domain is in the match
	return (minhash.chd.Get([]byte(forMatch)) != nil || minhash.chd.Get([]byte(rootdomain(forMatch))) != nil), nil
}