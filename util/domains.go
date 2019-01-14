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

func ReverseLookupDomainString(ipString string) string {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return ""
	}
	return ReverseLookupDomain(&ip)
}

// take a domain like "sub.main.top.tld" and transform to "tld.top.main.sub"
func ReverseDomainTree(domain string) string {
	domain = strings.TrimSpace(domain)
	if "" == domain || !strings.Contains(domain, ".") {
		return domain
	}

	if strings.HasSuffix(domain, ".") {
		domain = domain[:len(domain)-1]
	}

	// split into segments
	parts := strings.Split(domain, ".")

	// return domain if less than two parts are found
	if len(parts) < 2 {
		return domain
	}

	// start at the back of the parts with the builder and add to output string
	var builder strings.Builder
	for idx := len(parts) - 1; idx >= 0; idx-- {
		builder.WriteString(parts[idx])
		if idx != 0 {
			builder.WriteString(".")
		}
	}
	return builder.String()
}

// based on a domain name ("sub.google.com") returns a slice of domains that
// should be matched EXACTLY against ["sub.google.com", "google.com", "com"]
func DomainList(domain string) []string {
	if "" == strings.TrimSpace(domain) {
		return []string{}
	}

	domains := []string{}
	check := strings.ToLower(strings.TrimSpace(domain))
	for len(check) > 0 {
		domains = append(domains, check)
		split := strings.Split(check, ".")
		check = strings.Join(split[1:], ".")
	}
	return domains
}
