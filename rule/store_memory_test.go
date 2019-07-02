package rule

import (
	"testing"
)

func TestMemoryRuleStore(t *testing.T) {
	testStore(defaultRuleData, func() Store { return &memoryStore{} }, t)
}

func BenchmarkMemoryRuleStore(b *testing.B) {
	benchNonComplexStore(func() Store { return &memoryStore{} }, b)
}
