package util

import (
	"strings"

	"github.com/miekg/dns"
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

	// NXDOMAIN is pretty much empty
	if response.Rcode == dns.RcodeNameError {
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

// get the first A record response value
func GetFirstIPResponse(response *dns.Msg) string {
	if IsEmptyResponse(response) {
		return ""
	}

	for _, answer := range response.Answer {
		if aRecord, ok := answer.(*dns.A); ok {
			if aRecord != nil && aRecord.A != nil {
				return aRecord.A.String()
			}
		}
		if aaaaRecord, ok := answer.(*dns.AAAA); ok {
			if aaaaRecord != nil && aaaaRecord.AAAA != nil {
				return aaaaRecord.AAAA.String()
			}
		}
	}

	return ""
}

func GetAnswerValues(response *dns.Msg) []string {
	values := make([]string, 0)

	for _, rr := range response.Answer {
		value := GetRecordValue(rr)
		if "" != value {
			values = append(values, value)
		}
	}

	return values
}

// based on the string value for a RR
func GetRecordValue(record interface{}) string {

	var output string

	switch typed := record.(type) {
	// A
	case *dns.A:
		output = GetRecordValue(*typed)
	case dns.A:
		if typed.A != nil {
			output = typed.A.String()
		}
	// AAAA
	case *dns.AAAA:
		output = GetRecordValue(*typed)
	case dns.AAAA:
		if typed.AAAA != nil {
			output = typed.AAAA.String()
		}
	// generic catch-all for RR
	case dns.RR:
		output = typed.String()
	default:
		// no-op because "" is already the default string
	}

	return output
}
