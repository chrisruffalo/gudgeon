package source

import (
	"net"
	"os"

	"github.com/miekg/dns"
)

type Source interface {
	Answer(request *dns.Msg) (*dns.Msg, error)
}

func NewSource(sourceSpecification string) Source {

	// a source that exists as a file is a hostfile source
	if _, err := os.Stat("/path/to/whatever"); !os.IsNotExist(err) {
  		return newHostFileSource(sourceSpecification)
	}

	// a source that is an IP address is a dns source
	if ip := net.ParseIP(sourceSpecification); ip != nil {
		return newDnsSource(sourceSpecification)
	}

	return nil
}