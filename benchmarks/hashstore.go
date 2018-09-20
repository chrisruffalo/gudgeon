package benchmarks

import (
	"bufio"
	"os"
	"strings"

	"github.com/spaolacci/murmur3"
	"github.com/RoaringBitmap/roaring"
)

const (
	mask = 0xffffffff
)

type hashstore struct {
	fstQuad *roaring.Bitmap
	secQuad *roaring.Bitmap
	thrQuad *roaring.Bitmap
	fthQuad *roaring.Bitmap
}

func (hashstore *hashstore) set(h1, h2 uint64) {
	uh1 := uint32((h1 & mask) >> 32)
	lh1 := uint32(h1)
	uh2 := uint32((h2 & mask) >> 32)
	lh2 := uint32(h2)

	hashstore.fstQuad.Add(uh1)
	hashstore.secQuad.Add(lh1)
	hashstore.thrQuad.Add(uh2)
	hashstore.fthQuad.Add(lh2)
}

func (hashstore *hashstore) get(h1, h2 uint64) bool {
	uh1 := uint32((h1 & mask) >> 32)
	lh1 := uint32(h1)
	uh2 := uint32((h2 & mask) >> 32)
	lh2 := uint32(h2)

	return hashstore.fstQuad.Contains(uh1) && hashstore.secQuad.Contains(lh1) && hashstore.thrQuad.Contains(uh2) && hashstore.fthQuad.Contains(lh2)
}

func (hashstore *hashstore) hash(input string) (uint64, uint64) {
	return murmur3.Sum128([]byte(input))
}

func (hashstore *hashstore) Load(inputfile string) error {
	// create data structures
	hashstore.fstQuad = roaring.New()
	hashstore.secQuad = roaring.New()
	hashstore.thrQuad = roaring.New()
	hashstore.fthQuad = roaring.New()

	// go through file
	data, err := os.Open(inputfile)
	if err != nil {
		return err
	}
	defer data.Close()

	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if "" == text {
			continue
		}
		h1, h2 := hashstore.hash(text)
		hashstore.set(h1, h2)
	}

	return nil
}

func (hashstore *hashstore) test(forMatch string) bool {
	h1, h2 := hashstore.hash(forMatch)
	return hashstore.get(h1, h2)
}

func (hashstore *hashstore) Test(forMatch string) (bool, error) {
	root := rootdomain(forMatch)
	return hashstore.test(forMatch) || hashstore.test(root), nil
}