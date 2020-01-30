package rule

import (
	"testing"
)

func TestBloomRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() Store { return &bloomStore{} }, t)
}

func BenchmarkBloomRuleStore(b *testing.B) {
	benchNonComplexStore(func() Store { return &bloomStore{} }, b)
}

func TestBloomSqlRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() Store {
		return &bloomStore{
			backingStore:     &sqlStore{},
			defaultRuleCount: benchRules,
		}
	}, t)
}

func BenchmarkBloomSqlRuleStore(b *testing.B) {
	benchNonComplexStore(func() Store {
		return &bloomStore{
			backingStore:     &sqlStore{},
			defaultRuleCount: benchRules,
		}
	}, b)
}

func BenchmarkBloomHashRuleStore(b *testing.B) {
	benchNonComplexStore(func() Store {
		return &bloomStore{
			backingStore:     &hashStore{},
			defaultRuleCount: benchRules,
		}
	}, b)
}
