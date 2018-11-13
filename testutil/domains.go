package testutil

import (
	"math/rand"
	"time"
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