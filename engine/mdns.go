package engine

import (
	"fmt"
	"net"
	"strings"

	"github.com/miekg/dns"
	gocache "github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	questionString    = "_services._dns-sd._udp.local."
	mdnsAddressString = "224.0.0.251:5353"
)

var hostnameReadPriority = []string{"txt:fn", "txt:f", "txt:md", "name", "name6", "hostname", "hostname6"}

func MulticastMdnsQuery() {
	m := &dns.Msg{}
	m.Question = []dns.Question{
		{Name: questionString, Qtype: dns.TypeSRV, Qclass: dns.ClassINET},
		{Name: questionString, Qtype: dns.TypeTXT, Qclass: dns.ClassINET},
	}
	m.RecursionDesired = false

	var err error

	co := new(dns.Conn)
	if co.Conn, err = net.Dial("udp", mdnsAddressString); err != nil {
		return
	}
	defer co.Close()

	co.WriteMsg(m)
	log.Debug("Sent mDNS Multicast Query")
}

func MulticastMdnsListen(msgChan chan *dns.Msg, closeChan chan bool) {
	addr, err := net.ResolveUDPAddr("udp", mdnsAddressString)
	if err != nil {
		log.Errorf("Address resolve failed: %s\n", err)
		return
	}
	co := new(dns.Conn)
	if co.Conn, err = net.ListenMulticastUDP("udp", nil, addr); err != nil {
		log.Errorf("Listen multicast failed")
		return
	}
	// probably not needed but defer so it closes no matter what
	defer co.Close()

	// make query after open to start messages coming in
	MulticastMdnsQuery()

	// keep running after error?
	keeprunning := true

	// pipe messages to internal stop/start switch
	internalChan := make(chan *dns.Msg)
	go func() {
		for {
			msg, err := co.ReadMsg()
			if err != nil {
				if keeprunning {
					log.Debugf("Reading mDNS message: %s", err)
					continue
				} else {
					break
				}
			}
			internalChan <- msg
		}
		close(internalChan)
		log.Debugf("Shutdown mDNS connection")
	}()

	// loop that decides to read/forward messages or close listener
	for {
		select {
		case <-closeChan:
			keeprunning = false
			err := co.Close()
			if err != nil {
				log.Errorf("Could not close mDNS listener")
			}
			close(msgChan)
			closeChan <- true
			log.Debugf("Closed mDNS listener")
			return
		case msg := <-internalChan:
			if msgChan != nil && msg != nil {
				msgChan <- msg
			}
		}
	}
}

// parses entries out of records and adds them to the map when found
func parseEntries(entryMap map[string]string, entries []dns.RR) {
	for _, rr := range entries {
		switch rr.Header().Rrtype {
		case dns.TypeA:
			name := rr.Header().Name
			if "" != name {
				entryMap["hostname"] = name
				if value, ok := rr.(*dns.A); ok && value != nil && value.A != nil {
					entryMap["address"] = value.A.String()
				}
				if idx := strings.Index(name, "."); idx > -1 {
					shortname := name[:idx]
					if "" != shortname {
						if name, found := entryMap["name"]; found {
							if len(shortname) < len(name) {
								entryMap["name"] = shortname
							}
						} else {
							entryMap["name"] = shortname
						}
					}
				}
			}
		case dns.TypeAAAA:
			name := rr.Header().Name
			if "" != name {
				entryMap["hostname6"] = name
				if value, ok := rr.(*dns.AAAA); ok && value != nil && value.AAAA != nil {
					entryMap["address6"] = value.AAAA.String()
				}
				if idx := strings.Index(name, "."); idx > -1 {
					shortname := name[:idx]
					if "" != shortname {
						if name, found := entryMap["name6"]; found {
							if len(shortname) < len(name) {
								entryMap["name6"] = shortname
							}
						} else {
							entryMap["name6"] = shortname
						}
					}
				}
			}
		case dns.TypeTXT:
			if len(rr.(*dns.TXT).Txt) > 0 {
				for _, txt := range rr.(*dns.TXT).Txt {
					if idx := strings.Index(txt, "="); idx > -1 {
						entryMap["txt:"+txt[:idx]] = txt[idx+1:]
					} else {
						entryMap["txt"] = txt
					}
				}
			}
		}
	}
}

func ParseMulticastMessage(msg *dns.Msg) (map[string]string, error) {
	if len(msg.Answer) < 1 && len(msg.Extra) < 1 {
		return nil, fmt.Errorf("No messages found")
	}

	// map of values parsed
	parsed := make(map[string]string)

	parseEntries(parsed, msg.Answer)
	parseEntries(parsed, msg.Extra)

	if len(parsed) < 1 {
		return nil, fmt.Errorf("No messages parsed")
	}

	return parsed, nil
}

func CacheMulticastMessages(cache *gocache.Cache, msgChan chan *dns.Msg) {
	var err error
	var parsed map[string]string
	for msg := range msgChan {
		parsed, err = ParseMulticastMessage(msg)
		if err != nil || len(parsed) < 1 {
			continue
		}

		// if there's no address in the parsed values, move onto the next message
		_, ok4 := parsed["address"]
		_, ok6 := parsed["address6"]
		if !ok4 && !ok6 {
			continue
		}

		// show valid entry, may convert this to trace at some point
		//log.Infof("parsed entry: %+v", parsed)

		// since we have a parsed message let's do something with it
		addresses := []string{parsed["address"], parsed["address6"]}
		for _, addr := range addresses {
			if "" == addr {
				continue
			}

			// get the old message
			current, found := cache.Get(addr)
			if !found {
				continue
			}

			// we are only caching this type of value in this cache but
			// make sure it is the right type before continuing
			cmap, okConvert := current.(map[string]string)
			if !okConvert {
				continue
			}

			for key, cval := range cmap {
				cval = strings.TrimSpace(cval)

				// skip current value if they are empty or just "0"
				if cval == "" || cval == "0" {
					continue
				}

				// if the newly parsed information is shorter, add it
				if pval, inParsed := parsed[key]; inParsed && len(cval) < len(pval) {
					parsed[key] = cval
				} else {
					parsed[key] = cval
				}
			}
		}

		if len(parsed) < 1 {
			continue
		}

		// now that we've collected allllll of the data, cache it
		for _, addr := range addresses {
			if "" == addr {
				continue
			}
			cache.Set(addr, parsed, gocache.DefaultExpiration)
		}
	}
}

func ReadCachedHostname(cache *gocache.Cache, address string) string {
	// get the old message
	current, found := cache.Get(address)
	if !found {
		return ""
	}

	cmap, okConvert := current.(map[string]string)
	if !okConvert {
		return ""
	}

	// go through the priority-ordered list of possible hostnames and
	// load the first found (highest priority) key
	for _, key := range hostnameReadPriority {
		if value, found := cmap[key]; found && "" != value && "0" != value {
			return value
		}
	}

	return ""
}
