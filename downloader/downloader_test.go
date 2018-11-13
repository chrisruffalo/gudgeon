package downloader

import (
	"github.com/chrisruffalo/gudgeon/config"
	"io/ioutil"
	"os"
	"testing"
)

// shortcut method that downloads all lists
func downloadAll(t *testing.T, config *config.GudgeonConfig) error {
	// go through lists
	for _, list := range config.Lists {
		err := Download(config, list)
		if err != nil {
			t.Logf("Error downloading list=<< %s >>: %s", list.Name, err)
		}
	}

	return nil
}

func conf(t *testing.T, path string) *config.GudgeonConfig {
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

func TestDownloadAll(t *testing.T) {
	config := conf(t, "testdata/testconfig.yml")
	defer os.RemoveAll(config.Home)

	err := downloadAll(t, config)
	if err != nil {
		t.Errorf("Error during downloads: %s", err)
	}
}

func TestDownloadOverwrite(t *testing.T) {
	config := conf(t, "testdata/testconfig.yml")
	defer os.RemoveAll(config.Home)

	// download once
	err := downloadAll(t, config)
	if err != nil {
		t.Errorf("Error during downloads: %s", err)
	}

	// download again over top
	err = downloadAll(t, config)
	if err != nil {
		t.Errorf("Error during re-downloads: %s", err)
	}
}

func TestBadDownloadScheme(t *testing.T) {
	config := conf(t, "testdata/badurl.yml")
	defer os.RemoveAll(config.Home)

	err := downloadAll(t, config)
	if err != nil {
		t.Errorf("Got error during download: %s", err)
	}
}

func TestBadDownloadUrl(t *testing.T) {
	config := conf(t, "testdata/badconfig.yml")
	defer os.RemoveAll(config.Home)

	err := downloadAll(t, config)
	if err != nil {
		t.Errorf("Got error during download: %s", err)
	}
}
