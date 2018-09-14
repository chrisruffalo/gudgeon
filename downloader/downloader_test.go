package downloader

import (
	"io/ioutil"
	"os"
	"testing"
	"github.com/chrisruffalo/gudgeon/config"
)

// shortcut method that downloads all lists
func downloadAll(t *testing.T, config *config.GudgeonConfig) error {
	// go through lists
	for _, list := range config.Blocklists {
		err := Download(config, list)
		if err != nil {
			t.Logf("Error downloading list=<< %s >>: %s", list.Name, err)
		}
	}

	return nil
}

func tmpConf(conf *config.GudgeonConfig) string {
	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	// update config
	if conf.Paths == nil {
		conf.Paths = new(config.GudgeonPaths)
	}
	conf.Paths.Cache = dir

	return dir
}

func TestDownloadAll(t *testing.T) {
	// load config
	config, err := config.Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// set up temporary config dir
	dir := tmpConf(config)
	defer os.RemoveAll(dir)

	err = downloadAll(t, config)
	if err != nil {
		t.Errorf("Error during downloads: %s", err)
	}
}

func TestDownloadOverwrite(t *testing.T) {
	// load config
	config, err := config.Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// set up temporary config dir
	dir := tmpConf(config)
	defer os.RemoveAll(dir)

	// download once
	err = downloadAll(t, config)
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
	// load config
	config, err := config.Load("testdata/badconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// set up temporary config dir
	dir := tmpConf(config)
	defer os.RemoveAll(dir)

	err = downloadAll(t, config)
	if err != nil {
		t.Errorf("Got error during download: %s", err)
	}
}

func TestBadDownloadUrl(t *testing.T) {
	// load config
	config, err := config.Load("testdata/badconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// set up temporary config dir
	dir := tmpConf(config)
	defer os.RemoveAll(dir)

	err = downloadAll(t, config)
	if err != nil {
		t.Errorf("Got error during download: %s", err)
	}
}