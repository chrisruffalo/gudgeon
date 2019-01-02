package resolver

import (
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
)

type Resolver interface {
	Answer(request *dns.Msg) (*dns.Msg, error)
}

func NewResolver(configuredResolver *GudgeonResolver) *Resolver {
	return nil
}
