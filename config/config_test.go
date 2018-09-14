package config

import (
	"testing"
)

func TestLoad(t *testing.T) {

	config, err := Load("testdata/testconfig.yml")
	if err != nil {
		t.Errorf("Error opening test config: %s", err)
	}

	// make sure loaded correct amount of lists
	if len(config.Blocklists) !=5 {
		t.Errorf("Unexpected number of Blocklists encoutered %d", len(config.Blocklists))
	}

	// get the source for each blocklist item
	for _, item := range config.Blocklists {
		if nil == item {
			t.Errorf("Item is nil")
			continue
		}
		if "" == item.Source {
			
		}
		if "" == config.PathToList(item) {
			t.Errorf("Could not get path for list named: %s", item.Name)
		}
	}
}
