package downloader

import (
	"io/ioutil"
	"os"
	"testing"
	"github.com/chrisruffalo/gudgeon/config"
)

func TestDownloadAll(t *testing.T) {
	// load config
	config, err := config.Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")
	// remove tmp dir at the end
	defer os.RemoveAll(dir)

	// update config
	config.Paths.Cache = dir

	lines, err := DownloadAll(config)
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

func TestDownloadOverwrite(t *testing.T) {
	// load config
	config, err := config.Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Could not load test configuration: %s", err)
	}

	// create/get tmp dir
	dir, _ := ioutil.TempDir("", "gudgeon-cache-")

	// update config
	config.Paths.Cache = dir

	// download once
	_, err = DownloadAll(config)
	if err != nil {
		t.Errorf("Error during downloads: %s", err)
	}

	// download again over top
	lines, err := DownloadAll(config)
	if err != nil {
		t.Errorf("Error during re-downloads: %s", err)
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
	defer os.RemoveAll(dir)

	// update config
	config.Paths.Cache = dir

	_, err = DownloadAll(config)
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
	defer os.RemoveAll(dir)

	// update config
	config.Paths.Cache = dir

	_, err = DownloadAll(config)
	if err == nil {
		t.Errorf("Should have produced a bad protocol scheme error during download")
	}
}