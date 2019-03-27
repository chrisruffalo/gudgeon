package engine

import (
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
		dns.Question{Name: questionString, Qtype: dns.TypeSRV, Qclass: dns.ClassINET},
		dns.Question{Name: questionString, Qtype: dns.TypeTXT, Qclass: dns.ClassINET},
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

func MulticastMdnsListen(msgChan chan *dns.Msg) {
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
	defer co.Close()

	// make query after open
	MulticastMdnsQuery()

	for {
		msg, err := co.ReadMsg()
		if err != nil {
			log.Debugf("Reading mDNS message: %s", err)
			continue
		}
		if msgChan != nil && msg != nil {
			msgChan <- msg
		}
	}
}

func ParseMulticastMessage(msg *dns.Msg) map[string]string {
	// map of values parsed
	parsed := make(map[string]string)

	// collect all entries
	entries := []dns.RR{}
	entries = append(entries, msg.Answer...)
	entries = append(entries, msg.Extra...)

	for _, rr := range entries {
		switch rr.Header().Rrtype {
		case dns.TypeA:
			name := rr.Header().Name
			if "" != name {
				parsed["hostname"] = name
				if value, ok := rr.(*dns.A); ok && value != nil && value.A != nil {
					parsed["address"] = value.A.String()
				}
				if strings.Contains(name, ".") {
					shortname := strings.Split(name, ".")[0]
					if "" != shortname {
						if name, found := parsed["name"]; found {
							if len(shortname) < len(name) {
								parsed["name"] = shortname
							}
						} else {
							parsed["name"] = shortname
						}
					}
				}
			}
		case dns.TypeAAAA:
			name := rr.Header().Name
			if "" != name {
				parsed["hostname6"] = name
				if value, ok := rr.(*dns.AAAA); ok && value != nil && value.AAAA != nil {
					parsed["address6"] = value.AAAA.String()
				}
				if strings.Contains(name, ".") {
					shortname := strings.Split(name, ".")[0]
					if "" != shortname {
						if name, found := parsed["name6"]; found {
							if len(shortname) < len(name) {
								parsed["name6"] = shortname
							}
						} else {
							parsed["name6"] = shortname
						}
					}
				}
			}
		case dns.TypeTXT:
			if len(rr.(*dns.TXT).Txt) > 0 {
				for _, txt := range rr.(*dns.TXT).Txt {
					if split := strings.Split(txt, "="); len(split) > 1 {
						parsed["txt:"+split[0]] = split[1]
					} else {
						parsed["txt"] = txt
					}
				}
			}
		}
	}

	return parsed
}

func CacheMulticastMessages(cache *gocache.Cache, msgChan chan *dns.Msg) {
	for msg := range msgChan {
		parsed := ParseMulticastMessage(msg)
		// if there's no address in the parsed values, move onto the next message
		_, ok4 := parsed["address"]
		_, ok6 := parsed["address6"]
		if !ok4 && !ok6 {
			continue
		}

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

			cmap, okConvert := current.(map[string]string)
			if !okConvert {
				continue
			}

			// add current values into parsed value
			for key, cval := range cmap {
				if pval, inParsed := parsed[key]; inParsed {
					if "" != cval && "0" != cval && len(cval) < len(pval) {
						parsed[key] = cval
					}
				} else {
					parsed[key] = cval
				}
			}
		}

		if len(parsed) < 1 {
			continue
		}

		// now that we've collected allllll of the data, store it
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

	// go through the priority-ordered list of posible hostnames and
	// load the highest priority key
	for _, key := range hostnameReadPriority {
		if value, found := cmap[key]; found && "" != value {
			return value
		}
	}

	return ""
}
