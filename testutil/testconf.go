package testutil

import (
	"io/ioutil"
	"testing"

	"github.com/chrisruffalo/gudgeon/config"
)

func Conf(t *testing.T, path string) *config.GudgeonConfig {
	// create/get tmp dir
	dir := TempDir()

	conf, err := config.Load(path)
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// update home
	conf.Home = dir

	return conf
}

func TempDir() string {
	dir, err := ioutil.TempDir("", "gudgeon-cache-")
	if err != nil {
		return "./gudgeon-cache-"
	}
	return dir
}
