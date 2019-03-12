package resolver

import (
	"os"
	"strings"

	"github.com/miekg/dns"
	"github.com/ryanuber/go-glob"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	zoneWildPrefix = "*."
)

type zoneSource struct {
	filePath string
	// map[domain] -> map[class] -> map[rrtype] -> []rr
	records   map[string]map[uint16]map[uint16][]dns.RR
	wildnames []string
}

func newZoneSourceFromFile(zoneFile string) (Source, error) {
	// get reader for zone file
	file, err := os.Open(zoneFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	filePath := file.Name()

	// create zone parser
	zp := dns.NewZoneParser(file, "", filePath)
	if err := zp.Err(); err != nil {
		return nil, err
	}

	source := &zoneSource{
		filePath:  filePath,
		wildnames: make([]string, 0),
		records:   make(map[string]map[uint16]map[uint16][]dns.RR),
	}

	for rr, hasNext := zp.Next(); true; {
		if zp.Err() != nil {
			break
		}

		if rr != nil && rr.Header() != nil {
			name := strings.ToLower(rr.Header().Name)

			// record as a name that has a wildcard prefix
			if strings.HasPrefix(name, zoneWildPrefix) {
				source.wildnames = append(source.wildnames, name)
			}

			// add record
			source.addRecord(rr)

			// if it's an A record we can create a PTR record too
			if aRec, ok := rr.(*dns.A); ok && aRec.A != nil {
				ptr := &dns.PTR{
					Hdr: dns.RR_Header{Name: util.ReverseLookupDomain(&aRec.A), Rrtype: dns.TypePTR, Class: rr.Header().Class, Ttl: rr.Header().Ttl},
					Ptr: name,
				}
				source.addRecord(ptr)
			}

			if aaaaRec, ok := rr.(*dns.AAAA); ok && aaaaRec.AAAA != nil {
				ptr := &dns.PTR{
					Hdr: dns.RR_Header{Name: util.ReverseLookupDomain(&aaaaRec.AAAA), Rrtype: dns.TypePTR, Class: rr.Header().Class, Ttl: rr.Header().Ttl},
					Ptr: name,
				}
				source.addRecord(ptr)
			}
		}

		// break if no next element
		if !hasNext {
			break
		}
		rr, hasNext = zp.Next()
	}

	// get error from last parsed line
	err = zp.Err()

	if len(source.records) == 0 {
		return nil, err
	}

	return source, err
}

func (zoneSource *zoneSource) addRecord(rr dns.RR) {
	name := strings.ToLower(rr.Header().Name)
	rclass := rr.Header().Class
	rrtype := rr.Header().Rrtype

	if _, found := zoneSource.records[name]; !found {
		zoneSource.records[name] = make(map[uint16]map[uint16][]dns.RR)
	}
	if _, found := zoneSource.records[name][rclass]; !found {
		zoneSource.records[name][rclass] = make(map[uint16][]dns.RR)
	}
	if _, found := zoneSource.records[name][rclass][rrtype]; !found {
		zoneSource.records[name][rclass][rrtype] = make([]dns.RR, 0, 1)
	}

	// add record to records for name and type
	zoneSource.records[name][rclass][rrtype] = append(zoneSource.records[name][rclass][rrtype], rr)
}

func (zoneSource *zoneSource) Name() string {
	return "zonefile:" + zoneSource.filePath
}

func (zoneSource *zoneSource) resolveName(name string, qClass uint16, qType uint16, intoResponse *dns.Msg) {
	if domainRecords, found := zoneSource.records[name]; found {
		classResponses := make([]map[uint16][]dns.RR, 0)
		if qClass == dns.ClassANY {
			for _, value := range domainRecords {
				classResponses = append(classResponses, value)
			}
		} else {
			classResponses = append(classResponses, domainRecords[qClass])
		}

		rrs := make([]dns.RR, 0)
		for _, cr := range classResponses {
			if qType == dns.TypeANY {
				for _, v := range cr {
					rrs = append(rrs, v...)
				}
			} else {
				rrs = append(rrs, cr[qType]...)
			}
		}

		// append responses if responses were made
		if len(rrs) > 0 {
			intoResponse.Answer = append(intoResponse.Answer, rrs...)
		}
	}
}

func (zoneSource *zoneSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	if request == nil {
		return nil, nil
	}

	// get details from question
	question := request.Question[0]
	name := strings.ToLower(question.Name)
	qType := question.Qtype
	qClass := question.Qclass

	// create new response message
	response := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative: true,
			Opcode:        dns.OpcodeQuery,
		},
		Answer: make([]dns.RR, 0),
	}
	response.SetReply(request)

	// resolve CNAMES first
	if qType == dns.TypeA || qType == dns.TypeAAAA {
		zoneSource.resolveName(name, qClass, dns.TypeCNAME, response)
	}
	zoneSource.resolveName(name, qClass, qType, response)
	if len(response.Answer) < 1 {
		// check wildcards
		for _, wild := range zoneSource.wildnames {
			// if a wildcard matches, break and leave
			if glob.Glob(wild, name) {
				zoneSource.resolveName(wild, qClass, qType, response)
				if len(response.Answer) > 0 {
					break
				}
			}
		}
	}

	// normalize names to question name
	for _, rr := range response.Answer {
		rr.Header().Name = name
	}

	// if not nil or empty update the context
	if context != nil && !util.IsEmptyResponse(response) {
		// don't cache responses
		context.Stored = true

		// update source used
		context.SourceUsed = zoneSource.Name()
	}

	return response, nil
}
