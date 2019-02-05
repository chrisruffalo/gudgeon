package qlog

import (
	"testing"

	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestNewQueryLog(t *testing.T) {
	conf := testutil.Conf(t, "testdata/dbtest.yml")

	// create new query log
	qlog, err := New(conf)

	if err != nil {
		t.Errorf("Error during test: %s", err)
		return
	}

	if qlog == nil {
		t.Errorf("Query log nil but expected to be created")
	}
}
