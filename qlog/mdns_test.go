// +build realenvironment

package qlog

import (
	"context"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestMdns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgChan := make(chan *dns.Msg)
	go MulticastMdnsListen(msgChan)
	counter := 0
	go func() {
		MulticastMdnsQuery()
		for _ = range msgChan {
			counter++
		}
	}()

	<-ctx.Done()
	close(msgChan)

	if counter < 1 {
		t.Errorf("Did not see any mDNS/Avahi services")
	}
}
