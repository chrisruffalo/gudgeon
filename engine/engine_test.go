package engine

import (
	"os"
	"testing"

	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestBasicEngine(t *testing.T) {
	config := testutil.Conf(t, "testdata/simple.yml")
	defer os.RemoveAll(config.Home)

	// create engine from test config
	engine, err := New(config)
	if err != nil {
		t.Errorf("Could not create a new engine: %s", err)
	}

	// test engine against block data (should not be blocked)
	if engine.IsDomainBlocked("", "google.com") {
		t.Errorf("Domain 'google.com' should not be blocked but it is")
	}
	if !engine.IsDomainBlocked("", "2468.go2cloud.org") {
		t.Errorf("Domain '2468.go2cloud.org' should be blocked but it is not")
	}
}
