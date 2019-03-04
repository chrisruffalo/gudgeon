package util

import (
	"encoding/binary"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

// this is an experimental feature
// pulled from: https://github.com/jpillora/icmpscan
// i made some changes to make it a litte more readable
// and, more importantly, to set the outbound dns flags
// to 0 so this would work
func LookupNetBIOSName(address string) (string, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return "", nil
	}

	m := &dns.Msg{}
	m.SetQuestion("CKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA.", dns.TypeSRV)

	// forcibly set the flags to 0, for some reaon I couldn't figure a way
	// to do this in the dns library and this seems to be the best other shot
	// for doing it
	bytes, err := m.Pack()
	bytes[2] = 0
	bytes[3] = 0

	conn, err := net.DialTimeout("udp", ip.String()+":137", 2*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	// write message
	if _, err = conn.Write(bytes); err != nil {
		return "", err
	}

	// response is 512 bytes
	rbytes := make([]byte, 512)
	_, err = conn.Read(rbytes)
	if err != nil {
		return "", err
	}

	// read the Num Answer bytes
	answers := binary.BigEndian.Uint16(rbytes[6:8])
	if answers == 0 {
		return "", nil
	}

	// chop the first 12 bytes and then fast-forward to where the names are
	rbytes = rbytes[12:]
	offset := 0
	for rbytes[offset] != 0 { // we're looking for a zero terminated string here as a start point
		offset++
		if offset == len(rbytes) {
			return "", nil
		}
	}

	// no answer in this case
	rbytes = rbytes[offset+1:]
	if len(rbytes) < 12 {
		return "", nil
	}

	// get the count of the number of names
	names := rbytes[10]
	if names == 0 {
		return "", nil
	}

	// start looking for the end of the (zero terminated) string again
	rbytes = rbytes[11:]
	offset = 0
	for rbytes[offset] != 0 {
		offset++
		if offset == len(rbytes) {
			return "", nil
		}
	}

	// return from the start of where we are looking until the 0 termination
	return strings.TrimSpace(string(rbytes[:offset])), err
}
