package rule

import (
	"testing"
)

func TestBloomRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return &bloomStore{} }, t)
}

func BenchmarkBloomRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return &bloomStore{} }, b)
}

func TestBloomSqlRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore {
		return &bloomStore{
			backingStore:     &sqlStore{},
			defaultRuleCount: 1000000,
		}
	}, t)
}

func BenchmarkBloomSqlRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore {
		return &bloomStore{
			backingStore:     &sqlStore{},
			defaultRuleCount: benchRules,
		}
	}, b)
}
