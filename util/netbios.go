package util

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// response is set at 512 bytes
const _byteBufferSize = 512

var netBiosContextTime = 10 * time.Second
var netBiosDeadlineTime = 5 * time.Second

var bytesPool = sync.Pool{New: func() interface{} {
	return make([]byte, _byteBufferSize)
}}

// this is an experimental feature
// pulled from: https://github.com/jpillora/icmpscan
// i made some changes to make it a little more readable
// and, more importantly, to set the outbound dns flags
// to 0 so this would work
func LookupNetBIOSName(address string) (string, error) {
	ip := net.ParseIP(address)
	if ip == nil {
		return "", nil
	}

	m := &dns.Msg{}
	m.SetQuestion("CKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA.", 33)

	// forcibly set the flags to 0, for some reason I couldn't figure a way
	// to do this in the dns library and this seems to be the best other shot
	// for doing it
	bytes, err := m.Pack()
	bytes[2] = 0
	bytes[3] = 0

	// create a context that cancels the request after a timeout
	context, cancel := context.WithTimeout(context.Background(), netBiosContextTime)
	defer cancel()

	// dialer that fail after timeout
	dialer := &net.Dialer{
		Timeout: 2 * time.Second,
	}

	conn, err := dialer.DialContext(context, "udp", ip.String()+":137")
	if err != nil {
		return "", fmt.Errorf("Dialing: %s", err)
	}
	conn.SetDeadline(time.Now().Add(netBiosDeadlineTime))
	defer conn.Close()

	// write message
	if _, err = conn.Write(bytes); err != nil {
		return "", fmt.Errorf("Writing: %s", err)
	}

	// after read is done extend the deadline again
	conn.SetDeadline(time.Now().Add(netBiosDeadlineTime))

	// get response byte buffer from pool
	rbytes := bytesPool.Get().([]byte)
	defer bytesPool.Put(rbytes)

	_, err = conn.Read(rbytes)
	if err != nil {
		return "", fmt.Errorf("Reading: %s", err)
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
