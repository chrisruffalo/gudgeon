package rule

import (
	"testing"
)

func TestSqliteRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return &sqlStore{} }, t)
}

func BenchmarkSqliteRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return &sqlStore{} }, b)
}
