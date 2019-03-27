// +build realenvironment

package util

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestNetBios(t *testing.T) {
	name, err := LookupNetBIOSName("127.0.0.1")
	log.Infof("name: %s", name)
	if err != nil {
		t.Errorf("Got error while looking up netbios name: %s", err)
	}

	if "" == name {
		t.Errorf("Did not get any value for netbios name")
	}

}
