package rule

import (
	"testing"
)

func TestMemoryRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return CreateStore("mem") }, t)
}

func BenchmarkMemoryRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return CreateStore("mem") }, b)
}
