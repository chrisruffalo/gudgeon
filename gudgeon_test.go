// +build leaks

package main

import (
    "fmt"
    "os"
    "testing"

    "github.com/fortytw2/leaktest"
    "github.com/miekg/dns"

    "github.com/chrisruffalo/gudgeon/resolver"
    "github.com/chrisruffalo/gudgeon/testutil"
)

func TestGudgeonLeaks(t *testing.T) {
    defer leaktest.Check(t)()

    // test with full configuration
    config := testutil.Conf(t, "./gudgeon-full.yml")
    defer os.RemoveAll(config.Home)

    // disable all reverse lookpu functions (mainly because we don't care but also because of mdns/avahi)
    falsePtr := false
    config.QueryLog.ReverseLookup = &falsePtr
    config.QueryLog.MdnsLookup = &falsePtr
    config.QueryLog.NetbiosLookup = &falsePtr

    // don't log queries to stdout or file
    config.QueryLog.Stdout = &falsePtr
    config.QueryLog.File = ""

    // create new gudgeon
    gudgeon := NewGudgeon(config)
    gudgeon.Start()

    // create a new DNS source pointed directly at the local engine
    dnsSource := resolver.NewSource("127.0.0.1:5354")

    // make a dns request shell
    m := &dns.Msg{
        MsgHdr: dns.MsgHdr{
            Authoritative:     true,
            AuthenticatedData: true,
            RecursionDesired:  true,
            Opcode:            dns.OpcodeQuery,
        },
    }

    // make question parts
    m.Question = make([]dns.Question, 1)

    // starting queries
    fmt.Printf("Starting queries...\n")

    doneChan := make(chan bool)

    // make 20,000 random queries
    go func() {
        for c := 0; c < 200; c++ {
            // make new question
            m.Question[0] = dns.Question{Name: dns.Fqdn(testutil.RandomDomain()), Qtype: dns.TypeA, Qclass: dns.ClassINET}

            // ask question from source
            rCon := resolver.DefaultRequestContext()
            _, _ = dnsSource.Answer(rCon, nil, m)

            if (c % 10) == 0 {
                fmt.Printf("Completed %d queries\n", c)
            }
        }
        doneChan <-true
    }()

    // wait for queries to finish
    <-doneChan
    fmt.Printf("Queries completed.\n")

    // stop when done
    gudgeon.Shutdown()
}