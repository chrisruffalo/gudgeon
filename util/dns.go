package util

import (
	"github.com/miekg/dns"
	"strings"
)

// returns true if the response is "empty"
// nil response
// empty response (no answers or other sections)
// answers with no content
func IsEmptyResponse(response *dns.Msg) bool {
	// easiest/quickest way to say it's not available
	if nil == response {
		return true
	}

	// guard against basic issues
	if len(response.Answer) < 1 && len(response.Ns) < 1 && len(response.Extra) < 1 {
		return true
	}

	// check each bit of the parts to make sure **something** was returned
	for _, parts := range [][]dns.RR{response.Answer, response.Ns, response.Extra} {
		if len(parts) > 0 {
			for _, rr := range parts {
				if rr != nil && rr.Header() != nil && len(strings.TrimSpace(rr.Header().String())) < len(strings.TrimSpace(rr.String())) {
					return false
				}
			}
			// all parts must have some content or the response is empty (content/answers but no actual content inside it)
			// this is mainly because sometimes you'll get an empty A/AAAA answer wwith an SOA attached in the NS which
			// as far as we're concerend isn't really an answer to anything
			return true
		}
	}

	return true
}
