package util

import (
    "encoding/binary"
    "fmt"
    "net"
    "strings"
    "time"

    "github.com/miekg/dns"
)


// this is an experimental feature
// pulled from: https://github.com/jpillora/icmpscan
func LookupNetBIOSName(address string) (string, error) {
    ip := net.ParseIP(address)
    if ip == nil {
        return "", nil
    }

    m := dns.Msg{}
    m.SetQuestion("CKAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA.", 33)
    b, err := m.Pack()
    if err != nil {
        return "", err
    }
    conn, err := net.Dial("udp", ip.String()+":137")
    if err != nil {
        return "", err
    }
    defer conn.Close()
    conn.SetDeadline(time.Now().Add(2 * time.Second))
    if _, err := conn.Write(b); err != nil {
        return "", err
    }
    buff := make([]byte, 512)
    n, err := conn.Read(buff)
    if err != nil {
        return "", err
    }
    if n < 12 {
        return "", fmt.Errorf("no header")
    }
    b = buff[:n]
    if m.Id != binary.BigEndian.Uint16(b[0:2]) {
        return "", fmt.Errorf("id mismatch")
    }
    //==== headers
    // flags := binary.BigEndian.Uint16(b[2:4])
    // questions := binary.BigEndian.Uint16(b[4:6])
    answers := binary.BigEndian.Uint16(b[6:8])
    // authority := binary.BigEndian.Uint16(b[8:10])
    // additional := binary.BigEndian.Uint16(b[10:12])
    if answers == 0 {
        return "", fmt.Errorf("no answers")
    }
    //==== answers
    b = b[12:]
    offset := 0
    for b[offset] != 0 {
        offset++
        if offset == len(b) {
            return "", fmt.Errorf("too short")
        }
    }
    // hostname := string(b[:offset])
    b = b[offset+1:]
    if len(b) < 12 {
        return "", fmt.Errorf("no answer")
    }
    // rtype := binary.BigEndian.Uint16(b[:2])
    // rclass := binary.BigEndian.Uint16(b[2:4])
    // ttl := binary.BigEndian.Uint32(b[4:8])
    // len := binary.BigEndian.Uint16(b[8:10])
    names := b[10]
    if names == 0 {
        return "", fmt.Errorf("no names")
    }
    b = b[11:]
    offset = 0
    for b[offset] != 0 {
        offset++
        if offset == len(b) {
            return "", fmt.Errorf("too short")
        }
    }
    netbiosName := strings.TrimSpace(string(b[:offset]))
    // macAddress := offset + 2 and onwards
    return netbiosName, nil
}