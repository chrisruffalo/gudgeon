package engine

import (
	"fmt"
	"testing"
	"time"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestNewQueryLog(t *testing.T) {
	conf := testutil.TestConf(t, "testdata/dbtest.yml")

	// create new query log
	db, err := createEngineDB(conf)
	if err != nil {
		t.Errorf("Could not create test qlog DB")
		return
	}

	qlog, err := NewQueryLog(conf, db)

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
	conf := testutil.TestConf(t, "testdata/dbtest.yml")

	// create new query log
	db, err := createEngineDB(conf)
	if err != nil {
		t.Errorf("Could not create test qlog DB")
		return
	}

	// create new query log
	qlog, err := NewQueryLog(conf, db)
	if err != nil {
		t.Errorf("Error during qlog creation: %s", err)
		return
	}

	// create new recorder
	rec := &recorder{
		db:   db,
		qlog: qlog,
	}

	// this equals about half an hour at one query per second and is inserted back to back
	totalEntries := 86400 / 24 / 2
	for i := 0; i < totalEntries; i++ {
		// create message for sending to various endpoints
		msg := &InfoRecord{}
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
			msg.Match = rule.MatchBlock
			msg.MatchRule = "*"
			msg.MatchList = "testlist"
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
		rec.buffer(msg)
	}

	// flush waiting batch entries
	rec.flush()

	// query entries based on address
	query := &QueryLogQuery{
		Address: "192.168.0.2",
	}
	results, _ := qlog.Query(query)
	if len(results) != totalEntries/2 {
		t.Errorf("Address query returned unexpected results: %d but expected %d", len(results), totalEntries/2)
	}

	// query entries based on limit/skip
	query = &QueryLogQuery{
		Skip:  10,
		Limit: totalEntries / 4,
	}
	results, _ = qlog.Query(query)
	if len(results) != totalEntries/4 {
		t.Errorf("Limit query returned unexpected results: %d but expected %d", len(results), totalEntries/4)
	}

	// query rule matched entries
	ptrMatch := rule.MatchBlock
	query = &QueryLogQuery{
		Match: &ptrMatch,
	}
	results, _ = qlog.Query(query)
	if len(results) != totalEntries/4 {
		t.Errorf("Match query returned unexpected results: %d but expected %d", len(results), totalEntries/4)
	}

	// query by query type and rule matched with limit
	query = &QueryLogQuery{
		Match:       &ptrMatch,
		RequestType: "AAAA",
		Limit:       10,
	}
	results, _ = qlog.Query(query)
	if len(results) > 10 || len(results) < 1 {
		t.Errorf("Limited type query returned unexpected results: %d but expected %d", len(results), 10)
	}

	// query by request domain
	query = &QueryLogQuery{
		RequestDomain: "google.com.",
	}
	results, _ = qlog.Query(query)
	if len(results) != (totalEntries - totalEntries/20) {
		t.Errorf("Domain query returned unexpected results: %d but expected %d", len(results), totalEntries-totalEntries/20)
	}
	for _, result := range results {
		if result.RequestDomain != query.RequestDomain {
			t.Errorf("Expected domain did not match: %s != %s", result.RequestDomain, query.RequestDomain)
		}
	}

	// shut down the recorder
	go rec.shutdown()

	// stop query log
	qlog.Stop()
}

func TestTortureRecorder(t *testing.T) {
	conf := testutil.TestConf(t, "testdata/dbtest.yml")

	// create new query log
	db, err := createEngineDB(conf)
	if err != nil {
		t.Errorf("Could not create test qlog DB")
		return
	}

	// create new query log
	qlog, err := NewQueryLog(conf, db)
	if err != nil {
		t.Errorf("Error during qlog creation: %s", err)
		return
	}

	// create new recorder
	rec := &recorder{
		db:   db,
		qlog: qlog,
	}

	// on my home network we service about 100k queries per day, what happens
	// if we try and insert 2 days back-to-back?
	totalEntries := 200000
	for i := 0; i < totalEntries; i++ {
		// create message for sending to various endpoints
		msg := &InfoRecord{}
		if i%4 == 0 {
			msg.Address = "192.168.0.4"
		} else if i%3 == 0 {
			msg.Address = "192.168.0.3"
		} else if i%2 == 0 { // address shifts between multiple values
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
		msg.Request.Question[0] = dns.Question{Name: fmt.Sprintf("%d.google.com.", i), Qtype: dns.TypeA, Qclass: dns.ClassINET}
		if i%4 == 0 { // block one quarter of queries
			msg.Match = rule.MatchBlock
			msg.MatchRule = "*"
			msg.MatchList = "testlist"
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
		rec.buffer(msg)

		// wait for an interval every quarter of the way through
		if i%(totalEntries/4) == 0 {
			time.Sleep(rec.flushInterval)
		}
	}

	// flush waiting batch entries
	rec.flush()

	// shut down the recorder
	go rec.shutdown()

	// stop query log
	qlog.Stop()
}
