package config

import (
	"testing"
)

// test that the init function is providing non-null results everywhere
func TestInit(t *testing.T) {
	config := &GudgeonConfig{}
	config.verifyAndInit()

	if config.Metrics == nil {
		t.Errorf("Expected GudgeonMetrics block")
	}

	if config.QueryLog == nil {
		t.Errorf("Expected GudgeonQueryLog block")
	}
}
