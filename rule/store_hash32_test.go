package rule

import (
	"testing"
)

func TestHash32RuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return &hashStore32{} }, t)
}

func BenchmarkHash32RuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return &hashStore32{} }, b)
}

func TestHash32SqlRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore {
		return &hashStore32{
			delegate: &sqlStore{},
		}
	}, t)
}

func BenchmarkHash32SqlRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore {
		return &hashStore32{
			delegate: &sqlStore{},
		}
	}, b)
}
