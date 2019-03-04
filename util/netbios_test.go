// +build realenvironment

package util

import (
	"testing"

    //"github.com/grandcat/zeroconf"
)

func TestNetBios(t *testing.T) {
	//name, err := LookupNetBIOSName("127.0.0.1")
    name, err := LookupNetBIOSName("10.0.0.209")
    fmt.Printf("name: %s\n", name)
	if err != nil {
		t.Errorf("Got error while looking up netbios name: %s", err)
	}

	if "" == name {
		t.Errorf("Did not get any value for netbios name")
	}

}
