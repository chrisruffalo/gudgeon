package util

import (
	"github.com/miekg/dns"
)

// returns true if the response is "empty"
// nil response
// empty response (no answers or other sections)
// answers with no content
func IsEmptyResponse(response *dns.Msg) bool {
	if nil == response {
		return true
	}

	if len(response.Answer) < 1 && len(response.Ns) < 1 && len(response.Extra) < 1 {
		return true
	}

	// check each bit of the parts to make sure **something** was returned
	for _, parts := range [][]dns.RR{response.Answer, response.Ns, response.Extra} {
		if len(parts) > 0 {
			for _, rr := range parts {
				if rr != nil && rr.Header() != nil && len(rr.Header().String()) < len(rr.String()) {
					return false
				}
			}
		}
	}

	return true
}
