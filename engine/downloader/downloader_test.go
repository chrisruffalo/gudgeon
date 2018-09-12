package downloader

import (
	"io/ioutil"
	"testing"
	"github.com/chrisruffalo/gudgeon/config"
)

func TestDownload(t *testing.T) {
	// load config
	config, err := config.Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	lines, err := Download(dir, config.Blocklists)
	if err != nil {
		t.Errorf("Error during downloads: %s", err)
	}
	if lines < 1 {
		t.Errorf("No lines downloaded: %s", err)
	}
	if lines < 60000 { // conservative estimate of line sizes
		t.Errorf("Not enough lines downloaded (%d downloaded)", lines)
	}	
}

func TestBadDownloadScheme(t *testing.T) {
	// load config
	config, err := config.Load("testdata/badconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	_, err = Download(dir, config.Blocklists)
	if err == nil {
		t.Errorf("Should have produced a bad protocol scheme error during download")
	}
}

func TestBadDownloadUrl(t *testing.T) {
	// load config
	config, err := config.Load("testdata/badconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	_, err = Download(dir, config.Blocklists)
	if err == nil {
		t.Errorf("Should have produced a bad protocol scheme error during download")
	}
}