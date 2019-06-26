package resolver

import (
	"testing"
)

func TestResolvSource(t *testing.T) {
	// load
	source := &resolvSource{}
	source.Load("./testdata/test-resolv.conf")

	// query?
}
