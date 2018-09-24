package benchmarks

import (
 	"io/ioutil"
 	"log"
 	"os"
 	"path"
 	"strings"

    "github.com/dgraph-io/badger"
)

const (
	batchSize = 5000
)

type badgerstore struct {
	dir string
	db *badger.DB
}

func (badgerstore *badgerstore) Id() string {
	return "badger: '" + badgerstore.dir + "'"
}

func (badgerstore *badgerstore) Load(inputfile string, testdir string) error {
	// where the db goes
	dir := path.Join(testdir, "badgerdb")

	// redirect log framework output
	f, err := os.OpenFile(path.Join(testdir,"badger-output.log"), os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer f.Close()
	log.SetOutput(f)

	// set dir
	badgerstore.dir = dir

	// open db
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	// if the db exists straight open it and return
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		db, err := badger.Open(opts)
		if err != nil {
			return err
		}
  		badgerstore.db = db
		return nil
	}

	// if it does not exist open it and move on (which makes it get created and used on the next cycle)
	db, err := badger.Open(opts)	

	// go through file
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}
	array := strings.Split(string(content), "\r")

	for i := 0; i < len(array); i += batchSize {
		end := i + batchSize
		if end > len(array) {
			end = len(array)
		}
		err = db.Update(func(txn *badger.Txn) error {
			for _, item := range array[i:end] {
				item = strings.TrimSpace(item)
				if "" == item {
					continue
				}
	  			txn.Set([]byte(item), []byte("group1, group2, group3, group4"))
	  		}
	  		return nil
		})
	}
	badgerstore.db = db

	return nil
}

func check(txn *badger.Txn, forMatch string) (bool, error) {
	item, err := txn.Get([]byte(forMatch))
	if err != nil && err != badger.ErrKeyNotFound {
		return false, err
	}

	if item != nil {
		return true, nil
	}

	return false, nil
}

func (badgerstore *badgerstore) Test(forMatch string) (bool, error) {
	txn := badgerstore.db.NewTransaction(false)
	defer txn.Discard()

	resp, err := check(txn, forMatch)
	if err != nil {
		return false, err
	}

	if !resp {
		resp, err = check(txn, rootdomain(forMatch))
		if err != nil {
			return false, err
		}
	}

	return resp, nil
}

func (badgerstore *badgerstore) Teardown() error {
	badgerstore.db.Close()
	return nil
}