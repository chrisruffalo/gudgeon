package util

import (
	"encoding/hex"
	"net"
	"strconv"
	"strings"
)

const (
	ipv4Arpa = ".in-addr.arpa."
	ipv6Arpa = ".ip6.arpa."
)

// finds the subdomain of a requested domain, so "www.google.com" returns "google.com" and "google.com" returns "google.com"
func SubDomain(domain string) string {
	split := strings.Split(domain, ".")
	if len(split) >= 2 {
		return strings.Join(split[1:], ".")
	}
	return domain
}

// finds the "root" domain, that is a the domain with just the name and the TLD
func RootDomain(domain string) string {
	split := strings.Split(domain, ".")
	if len(split) >= 2 {
		return strings.Join(split[len(split)-2:], ".")
	}
	return domain
}

// returns the reverse lookup arpa domain for the given IP
func ReverseLookupDomain(ip *net.IP) string {
	if ip == nil {
		return ""
	}
	bytes := *ip

	// create string builder
	var sb strings.Builder

	suffix := ipv4Arpa
	if ip.To4() == nil {
		suffix = ipv6Arpa
		ipChars := []rune(hex.EncodeToString(bytes))
		for idx := len(ipChars) - 1; idx >= 0; idx-- {
			sb.WriteString(string(ipChars[idx]))
			if idx > 0 {
				sb.WriteString(".")
			}
		}
	} else {
		// walk through ip bytes in reverse
		size := 4
		floor := len(bytes) - size
		for idx := len(bytes) - 1; idx >= floor; idx-- {
			sb.WriteString(strconv.Itoa(int(bytes[idx])))
			if idx > floor {
				sb.WriteString(".")
			}
		}
	}

	return sb.String() + suffix
}
