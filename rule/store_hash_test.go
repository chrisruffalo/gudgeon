package rule

import (
	"testing"
)

func TestHashRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return CreateStore("hash") }, t)
}

func BenchmarkHashRuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return CreateStore("hash") }, b)
}
