package rule

import (
	"testing"
)

func TestHash32RuleStore(t *testing.T) {
	testStore(defaultRuleData, func() RuleStore { return CreateStore("hash32") }, t)
}

func BenchmarkHash32RuleStore(b *testing.B) {
	benchNonComplexStore(func() RuleStore { return CreateStore("hash32") }, b)
}
