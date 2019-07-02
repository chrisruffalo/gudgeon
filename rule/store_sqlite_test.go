package rule

import (
	"testing"

	"github.com/fortytw2/leaktest"
)

func TestSqliteRuleStore(t *testing.T) {
	defer leaktest.Check(t)()
	testStore(defaultRuleData, func() Store { return &sqlStore{} }, t)
}

func BenchmarkSqliteRuleStore(b *testing.B) {
	benchNonComplexStore(func() Store { return &sqlStore{} }, b)
}
