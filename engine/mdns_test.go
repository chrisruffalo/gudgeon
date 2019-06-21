// +build realenvironment

package engine

import (
	"context"
	"github.com/sirupsen/logrus"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestMdns(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	msgChan := make(chan *dns.Msg)
	go MulticastMdnsListen(msgChan, make(chan bool))
	counter := 0
	go func() {
		MulticastMdnsQuery()
		for msg := range msgChan {
			logrus.Infof("Got DNS message: %v", msg.String())
			counter++
		}
	}()

	<-ctx.Done()
	close(msgChan)

	if counter < 1 {
		t.Errorf("Did not see any mDNS/Avahi services")
	}
}
