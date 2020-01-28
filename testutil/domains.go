package testutil

import (
	"math/rand"
	"net"
	"time"

	"github.com/chrisruffalo/gudgeon/util"
)

// all tlds must be 4 characters
var tlds = [][]byte{[]byte(".com"), []byte(".org"), []byte(".net")}
var characters = []byte("abcdefghijklmnopqrstuvwxyz")

func init() {
	rand.Seed(time.Now().UnixNano())
}

/**
 * RandomDomain - produces a random domain, with a random domain name and one of a few fixed TLDs
 * This function has been optimized as much as is possible to produce a result without requiring
 * much copying or otherwise over-allocating. This speeds up the execution of the tests as well
 * as reduces the amount of extra garbage in profiling reports.
 */
const (
	randPartDomainLen = 10
	maxDomainLen = 19
)
func RandomDomain() string {
	domainLen := rand.Intn(randPartDomainLen) + (maxDomainLen - randPartDomainLen)
	domain := make([]byte, maxDomainLen)
	var idx int
	for idx = 0; idx < domainLen - 4;idx++ {
		domain[idx] = characters[rand.Intn(len(characters))]
	}
	tldi := rand.Intn(len(tlds))
	for idx = 0; idx < len(tlds[tldi]);idx ++ {
		domain[domainLen - 4 + idx] = tlds[tldi][idx]
	}
	return util.ByteSliceToString(domain[:domainLen])
}

func ReverseIpString(ip string) string {
	ipObj := net.ParseIP(ip)
	return util.ReverseLookupDomain(&ipObj)
}
