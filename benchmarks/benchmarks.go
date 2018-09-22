package benchmarks

import (
	"strings"
)

type Benchmark interface {
	Id() string
	Load(inputfile string) error
	Test(forMatch string) (bool, error)
	Teardown() error
}

func rootdomain(domain string) string {
	split := strings.Split(domain, ".") 
	if len(split) >= 2 {
		return strings.Join(split[len(split)-2:], ".")
	}
	return domain
}

