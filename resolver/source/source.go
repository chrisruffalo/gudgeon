package source

import (
	"github.com/miekg/dns"
)

type Source interface {
	Answer(request *dns.Msg) (*dns.Msg, error)
}

