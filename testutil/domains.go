package testutil

import (
	"math/rand"
	"net"
	"time"

	"github.com/chrisruffalo/gudgeon/util"
)

var tlds = []string{".com", ".org", ".net"}
var runes = []rune("abcdefghijklmnopqrstuvwxyz")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandomDomain() string {
	domainLen := rand.Intn(10) + 5
	domain := make([]rune, domainLen)
	for idx := range domain {
		domain[idx] = runes[rand.Intn(len(runes))]
	}
	return string(domain) + tlds[rand.Intn(len(tlds))]
}

func ReverseIpString(ip string) string {
	ipObj := net.ParseIP(ip)
	return util.ReverseLookupDomain(&ipObj)
}
