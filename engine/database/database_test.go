package database

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"github.com/willf/bloom"
)


func TestInsertCheckCache(t *testing.T) {
	data := []struct {
		question string
		domain string
		answer string
		ttl int
	} {
		{ "A", "google.com", "172.217.5.238", 0},
		{ "AAAA", "google.com", "::ff", 0},
		{ "TXT", "google.com", "1.1.1.1", 0},
	}

	tempfile, err := ioutil.TempFile("", "gudgeon.*.db")
	if err != nil {
		t.Errorf("Unable to create tempfile: %s", err)
		return 
	}
	defer os.Remove(tempfile.Name())

	db, err := Get(tempfile.Name())
	if err != nil {
		t.Errorf("Unable to create database: %s", err)
		return
	}
	defer db.Close()

	for _, item := range data {
		err = db.InsertCache(item.question, item.domain, item.answer, item.ttl)
		if err != nil {
			t.Errorf("Unable to insert cache item: %s", err)
		}
		answer, err := db.CheckCache(item.question, item.domain)
		if err != nil {
			t.Errorf("Unable to check cache item: %s", err)	
		}
		if answer != item.answer {
			t.Errorf("Cached answer (%s) did not match inserted answer (%s)", answer, item.answer)
		}
	}
}

func BenchmarkInsertCache(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/domains.list")

	tempfile, err := ioutil.TempFile("", "gudgeon-benchmark-insert-cache.*.db")
	if err != nil {
		b.Errorf("Unable to create tempfile: %s", err)
		return 
	}
	defer os.Remove(tempfile.Name())

	db, err := Get(tempfile.Name())
	if err != nil {
		b.Errorf("Unable to create database: %s", err)
		return
	}
	defer db.Close()

	question, answer, ttl := "A", "0.0.0.0", 3600

	b.ResetTimer()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for i := 0; i < b.N && scanner.Scan(); i++ {
		err = db.InsertCache(question, scanner.Text(), answer, ttl)
		if err != nil {
			b.Errorf("Could not insert cache item: %s", err)
		}
	}
}

func TestInsertInactiveRule(t *testing.T) {

	data := []struct {
		block string
		blockType int
		listName string
		listType int
		tags []string
	} {
		{ "google.com", RuleTypeMatch, "standard", ListTypeWhite, []string{"default", "none"} },
		{ "google.com", RuleTypeMatch, "never", ListTypeBlack, []string{"default", "none"} },
		{ "google.com", RuleTypeWild, "blocker", ListTypeBlock, []string{"default", "none"} },
		{ "reddit.com", RuleTypeRegex, "standard", ListTypeWhite, []string{"default"} },
		{ "yahoo.com", RuleTypeRegex, "standard", ListTypeBlack, []string{"default","advert"} },
		{ "advert.com", RuleTypeRegex, "standard", ListTypeBlack, []string{"default","advert"} },
		{ "advert.com", RuleTypeMatch, "other", ListTypeWhite, []string{"default","advert"} },
		{ "google.com", RuleTypeMatch, "standard", ListTypeBlack, []string{"default"} },
		{ "advert.com", RuleTypeWild, "standard", ListTypeBlock, []string{"default","strict","advert","nobody"} },
	}

	tempfile, err := ioutil.TempFile("", "gudgeon.*.db")
	if err != nil {
		t.Errorf("Unable to create tempfile: %s", err)
		return 
	}
	defer os.Remove(tempfile.Name())

	db, err := Get(tempfile.Name())
	if err != nil {
		t.Errorf("Unable to create database: %s", err)
		return
	}
	defer db.Close()

	for _, item := range data {
		err = db.InsertInactiveRule(item.block, item.blockType, item.listName, item.listType, item.tags)
		if err != nil {
			t.Errorf("Unable to insert rule %s to list %s of type %d (tags: %s) with error: %s", item.block, item.listName, item.listType, item.tags, err)
		}
	}
}

func BenchmarkInsertRule(b *testing.B) {
	data, err := ioutil.ReadFile("testdata/domains.list")

	tempfile, err := ioutil.TempFile("", "gudgeon-benchmark-insert-rule.*.db")
	if err != nil {
		b.Errorf("Unable to create tempfile: %s", err)
		return 
	}
	defer os.Remove(tempfile.Name())

	db, err := Get(tempfile.Name())
	if err != nil {
		b.Errorf("Unable to create database: %s", err)
		return
	}
	defer db.Close()

	blockType, listName, listType, tags := RuleTypeMatch, "default", ListTypeWhite, []string{"default", "testmark", "bulk"}

	b.ResetTimer()
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for i := 0; i < b.N && scanner.Scan(); i++ {
		err = db.InsertInactiveRule(scanner.Text(), blockType, listName, listType, tags)
		if err != nil {
			b.Errorf("Unable to insert rule: %s", err)
		}
	}
}

// func BenchmarkBulkInsertRule(b *testing.B) {
// 	tempfile, err := ioutil.TempFile("", "gudgeon-benchmark-insert-rule.*.db")
// 	if err != nil {
// 		b.Errorf("Unable to create tempfile: %s", err)
// 		return 
// 	}
// 	//defer os.Remove(tempfile.Name())

// 	db, err := Get(tempfile.Name())
// 	if err != nil {
// 		b.Errorf("Unable to create database: %s", err)
// 		return
// 	}
// 	defer db.Close()

// 	listName, listType, tags := "default", ListTypeWhite, []string{"default", "testmark", "bulk"}

// 	b.ResetTimer()
// 	file, err := os.Open("testdata/domains.list")
// 	if err != nil {
// 		b.Errorf("Unable to open test data file: %s", err)
// 	}
// 	err = db.BulkInsertRule(file, listName, listType, tags, false)
// 	if err != nil {
// 		b.Errorf("Unable to insert bulk rules: %s", err)
// 	}

// }

func BenchmarkRawScan(b *testing.B) {

	data, err := ioutil.ReadFile("testdata/domains.list")
	if err != nil {
		b.Errorf("Unable to read test data file: %s", err)
	}

	checkString := "google.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		result := false
		for scanner.Scan() {
			result = (scanner.Text() == checkString) || result
		}
		if !result {
			b.Errorf("failed to match: %s", checkString)
		}
	}
}

func BenchmarkRawFileScan(b *testing.B) {

	checkString := "google.com"

	for i := 0; i < b.N; i++ {
		file, err := os.Open("testdata/domains.list")
		if err != nil {
			b.Errorf("Unable to read test data file: %s", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		result := false
		for scanner.Scan() {
			result = (scanner.Text() == checkString) || result
		}
		if !result {
			b.Errorf("failed to match: %s", checkString)
		}
	}
}

func BenchmarkBloomFilter(b *testing.B) {
	file, err := os.Open("testdata/domains.list")
	if err != nil {
		b.Errorf("Unable to open test data file: %s", err)
	}
	defer file.Close()

	// create bloom filter
	load := uint(1000000)
	factor := uint(25)
	keys := uint(10)
	filter := bloom.New(load * factor, keys)

	// load bloom filter
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		filter.Add([]byte(scanner.Text()))
	}

	checkString := "ruffalo.org"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if filter.Test([]byte(checkString)) {
			b.Errorf("should not have matched: %s", checkString)
		}
	}
}