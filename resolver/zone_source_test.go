package resolver

import (
	"testing"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/testutil"
)

func TestLoadZoneFile(t *testing.T) {
	zoneSource := &zoneSource{}
	zoneSource.Load("./testdata/zone-test.db")
}

func TestZoneSource(t *testing.T) {

	data := []struct {
		name     string
		qclass   uint16
		qtype    uint16
		expected int
	}{
		{"dns1.greenstream.coop.", dns.ClassINET, dns.TypeA, 1},
		{"dns1.greenstream.coop.", dns.ClassINET, dns.TypeAAAA, 1},
		{"greenstream.coop.", dns.ClassINET, dns.TypeANY, 9},
		{"greenstream.coop.", dns.ClassANY, dns.TypeANY, 9},
		{"greenstream.coop.", dns.ClassCSNET, dns.TypeANY, 0},
		{"cloud.greenstream.coop.", dns.ClassINET, dns.TypeA, 1},
		{"cloud.greenstream.coop.", dns.ClassINET, dns.TypeAAAA, 0},
		{"app.cloud.greenstream.coop.", dns.ClassINET, dns.TypeA, 3},
		{"web.cloud.greenstream.coop.", dns.ClassINET, dns.TypeA, 3},
		{"app.cloud.greenstream.coop.", dns.ClassINET, dns.TypeAAAA, 2},
		{"web.cloud.greenstream.coop.", dns.ClassINET, dns.TypeAAAA, 2},
		{testutil.ReverseIpString("42.112.115.45"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("42.112.115.45"), dns.ClassINET, dns.TypeA, 0},
		{testutil.ReverseIpString("42.112.115.46"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("42.112.115.47"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("42.112.115.48"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("42.112.115.49"), dns.ClassINET, dns.TypePTR, 0},
		{testutil.ReverseIpString("de3d:b3ef:2385:a::a"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("de3d:b3ef:2385:a::b"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("de3d:b3ef:2385:a::c"), dns.ClassINET, dns.TypePTR, 1},
		{testutil.ReverseIpString("de3d:b3ef:2385:a::d"), dns.ClassINET, dns.TypePTR, 0},
	}

	zone := &zoneSource{}
	zone.Load("./testdata/zone-test.db")

	for _, d := range data {
		// create question
		request := &dns.Msg{
			MsgHdr: dns.MsgHdr{
				Authoritative: true,
				Opcode:        dns.OpcodeQuery,
			},
			Question: []dns.Question{
				{Name: d.name, Qtype: d.qtype, Qclass: d.qclass},
			},
		}

		// make question
		response, err := zone.Answer(nil, nil, request)

		if err != nil {
			t.Errorf("Error asking question: %s", err)
			continue
		}

		if (response == nil || response.Answer == nil) && d.expected != 0 {
			t.Errorf("Nil answers not expected for question '%s'", d.name)
			continue
		} else if (response == nil || response.Answer == nil) && d.expected == 0 {
			continue
		}

		if len(response.Answer) != d.expected {
			t.Errorf("Query '%s' did not get %d responses, got %d", d.name, d.expected, len(response.Answer))
			continue
		}

		//t.Logf("Query '%s' returned expected result: %d", d.name, d.expected)
	}

}
