package rule

import (
    "testing"
)

func TestSqliteRuleStore(t *testing.T) {
    testStore(defaultRuleData, func() RuleStore { return CreateStore("sqlite") }, t)
}

func BenchmarkSqliteRuleStore(b *testing.B) {
    benchNonComplexStore(func() RuleStore { return CreateStore("sqlite") }, b)
}
