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
				if rr != nil && rr.Header() != nil && rr.Header().Rrtype != dns.TypeNone && !IsRecordEmpty(rr) {
					return false
				}
			}
			// all parts must have some content or the response is empty (content/answers but no actual content inside it)
			// this is mainly because sometimes you'll get an empty A/AAAA answer wwith an SOA attached in the NS which
			// as far as we're concerned isn't really an answer to anything
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
	// return empty list if answer values are not present
	if response == nil || len(response.Answer) < 1 {
		return []string{}
	}

	values := make([]string, 0, len(response.Answer))

	var value string
	for i := 0; i < len(response.Answer); i++ {
		value = GetRecordValue(response.Answer[i])
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
		if typed.A != nil {
			output = typed.A.String()
		}
	case dns.A:
		if typed.A != nil {
			output = typed.A.String()
		}
	// AAAA
	case *dns.AAAA:
		if typed.AAAA != nil {
			output = typed.AAAA.String()
		}
	case dns.AAAA:
		if typed.AAAA != nil {
			output = typed.AAAA.String()
		}
	// PTR
	case *dns.PTR:
		output = typed.Ptr
	case dns.PTR:
		output = typed.Ptr
	// TXT
	case *dns.TXT:
		output = strings.Join(typed.Txt, " ")
	case dns.TXT:
		output = strings.Join(typed.Txt, " ")
	// generic catch-all for RR
	case dns.RR:
		output = dns.TypeToString[typed.Header().Rrtype] + "= " + typed.String()
	default:
		// no-op because "" is already the default string
	}

	return output
}

func IsRecordEmpty(record interface{}) bool {
	switch typed := record.(type) {
	// A
	case *dns.A:
		return typed.A == nil
	case dns.A:
		return typed.A == nil
	// AAAA
	case *dns.AAAA:
		return typed.AAAA == nil
	case dns.AAAA:
		return typed.AAAA == nil
	// PTR
	case *dns.PTR:
		return typed.Ptr == ""
	case dns.PTR:
		return typed.Ptr == ""
	// TXT
	case *dns.TXT:
		return len(typed.Txt) == 0
	case dns.TXT:
		return len(typed.Txt) == 0
	// CNAME
	case *dns.CNAME:
		return "" == typed.Target
	case dns.CNAME:
		return "" == typed.Target
	// SOA
	case *dns.SOA:
		return "" == typed.Ns
	case dns.SOA:
		return "" == typed.Ns
	// generic catch-all for RR
	case dns.RR:
		return len(typed.Header().String()) >= len(typed.String())
	default:
		// no-op because "" is already the default string
	}

	return true
}
