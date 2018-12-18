package testutil

import (
	"io/ioutil"
	"testing"

	"github.com/chrisruffalo/gudgeon/config"
)

func Conf(t *testing.T, path string) *config.GudgeonConfig {
	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	conf, err := config.Load(path)
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// update home
	conf.Home = dir

	return conf
}