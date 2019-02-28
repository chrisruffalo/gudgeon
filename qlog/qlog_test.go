package qlog

import (
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestNewQueryLog(t *testing.T) {
	conf := testutil.Conf(t, "testdata/dbtest.yml")

	// create new query log
	qlog, err := New(conf)

	if err != nil {
		t.Errorf("Error during qlog creation: %s", err)
		return
	}

	if qlog == nil {
		t.Errorf("Query log nil but expected to be created")
	}

	// stop query log
	qlog.Stop()
}

func TestQueryLogQuery(t *testing.T) {
	conf := testutil.Conf(t, "testdata/dbtest.yml")

	// create new query log
	qlogInterface, err := New(conf)
	if err != nil {
		t.Errorf("Error during qlog creation: %s", err)
		return
	}

	qlog := qlogInterface.(*qlog)

	// log 1000 entries
	totalEntries := 86400 / 24 // about one hour at one query per second
	for i := 0; i < totalEntries; i++ {
		// create message for sending to various endpoints
		msg := new(LogInfo)
		if i%2 == 0 { // address shifts between two values
			msg.Address = "192.168.0.2"
		} else {
			msg.Address = "192.168.0.1"
		}
		msg.Request = &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative:     true,
				AuthenticatedData: true,
				RecursionDesired:  true,
				Opcode:            dns.OpcodeQuery,
			},
		}
		msg.Request.Question = make([]dns.Question, 1)
		msg.Request.Question[0] = dns.Question{Name: "google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
		if i%4 == 0 { // block one quarter of queries
			msg.Blocked = true
			msg.BlockedRule = "*"
			msg.BlockedList = "testlist"
		}
		if i%20 == 0 {
			msg.RequestDomain = "netflix.com."
		} else {
			msg.RequestDomain = "google.com."
		}
		if i%10 == 0 {
			msg.RequestType = "AAAA"
		} else {
			msg.RequestType = "A"
		}
		msg.Response = &dns.Msg{}
		msg.Response.SetReply(msg.Request)
		msg.Result = &resolver.ResolutionResult{}
		msg.RequestContext = &resolver.RequestContext{}
		msg.Created = time.Now()

		// log msg
		qlog.logDB(msg, false)
	}

	// flush waiting batch entries
	qlog.logDB(nil, true)

	// query entries based on address
	query := &QueryLogQuery{
		Address: "192.168.0.2",
	}
	results := qlog.Query(query)
	if len(results) != totalEntries/2 {
		t.Errorf("Address query returned unexpected results: %d but expected %d", len(results), totalEntries/2)
	}

	// query entries based on limit/skip
	query = &QueryLogQuery{
		Skip:  10,
		Limit: totalEntries / 4,
	}
	results = qlog.Query(query)
	if len(results) != totalEntries/4 {
		t.Errorf("Limit query returned unexpected results: %d but expected %d", len(results), totalEntries/4)
	}

	// query blocked entries
	ptrTrue := true
	query = &QueryLogQuery{
		Blocked: &ptrTrue,
	}
	results = qlog.Query(query)
	if len(results) != totalEntries/4 {
		t.Errorf("Blocked query returned unexpected results: %d but expected %d", len(results), totalEntries/4)
	}

	// query by query type and blocked with limit
	query = &QueryLogQuery{
		Blocked:     &ptrTrue,
		RequestType: "AAAA",
		Limit:       10,
	}
	results = qlog.Query(query)
	if len(results) > 10 || len(results) < 1 {
		t.Errorf("Limited type query returned unexpected results: %d but expected %d", len(results), 10)
	}

	// query by request domain
	query = &QueryLogQuery{
		RequestDomain: "google.com.",
	}
	results = qlog.Query(query)
	if len(results) != (totalEntries - totalEntries/20) {
		t.Errorf("Domain query returned unexpected results: %d but expected %d", len(results), totalEntries-totalEntries/20)
	}
	for _, result := range results {
		if result.RequestDomain != query.RequestDomain {
			t.Errorf("Expected domain did not match: %s != %s", result.RequestDomain, query.RequestDomain)
		}
	}

	// stop query log
	qlog.Stop()
}