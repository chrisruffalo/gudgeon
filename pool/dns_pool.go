package pool

import (
	"crypto/tls"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"
	"net"
	"sync"
	"time"
)

/**
 * This is meant to be a connection pool implementation that tracks persistent connections
 * and returns new connections. For non-persistent connections (UDP) it should still block
 * until a connection is available even though the connection should not be returned
 */

const (
	DefaultMaxConnections int = 2
	DefaultDeadline           = 200 * time.Millisecond
)

type DnsPoolConfiguration struct {
	// max connections this pool can hand out
	MaxConnections int
}

var DefaultDnsPoolConfiguration = DnsPoolConfiguration{
	MaxConnections: DefaultMaxConnections,
}

// ConnectionPool interface for reusing existing and getting new connections
type DnsPool interface {
	// thread safe, will block until a connection is available
	// creates new connections until the pool is full
	Get() (*dns.Conn, error)
	// return a connection to the pool and handle network appropriate actions
	// a tcp connection will be retained, a udp connection will be closed and not reused
	// this is thread-safe but not for a given connection (which is ok because two threads probably shouldn't try and release the same connection at the same time)
	Release(conn *dns.Conn)
	// discard a connection and remove it from the pool (used for error'd connections)
	// this is thread-safe but not for a given connection (which is ok because two threads probably shouldn't try and discard the same connection at the same time)
	Discard(conn *dns.Conn)
	// shutdown enables a clean/safe shutdown of the pool
	Shutdown()
}

type dnsPool struct {
	// if the pool is shutting down then don't hand out connections at all
	shutdownMtx sync.Mutex
	shutdown    bool

	// dialer
	dialer net.Dialer

	// tls configuration
	tlsConfig tls.Config

	// where the pool dials to
	protocol string
	network  string
	host     string
	port     int
	address  string

	// the initial pool configuration
	config DnsPoolConfiguration

	// channel for connection cycling
	cons chan net.Conn
}

func DefaultDnsPool(protocol string, address string) DnsPool {
	return NewDnsPool(protocol, address, DefaultDnsPoolConfiguration)
}

func NewDnsPool(protocol string, address string, config DnsPoolConfiguration) DnsPool {
	// create new connection pool
	conPool := &dnsPool{
		shutdown: false,
		protocol: protocol,
		address:  address,
		config:   config,
	}

	if protocol == "tcp-tls" {
		conPool.network = "tcp"
	} else {
		conPool.network = protocol
	}

	// configure tls
	conPool.tlsConfig = tls.Config{InsecureSkipVerify: true}

	// create dialer
	// keep dialer for reuse
	conPool.dialer = net.Dialer{}
	// set tcp dialer properties
	if protocol == "tcp" || protocol == "tcp-tls" {
		conPool.dialer.KeepAlive = 0
	}
	conPool.dialer.Timeout = DefaultDeadline

	// create channel of requested size
	conPool.cons = make(chan net.Conn, config.MaxConnections)

	// fill with empty connections
	for c := 0; c < config.MaxConnections; c++ {
		conPool.cons <- nil
	}

	return conPool
}

func (pool dnsPool) Get() (*dns.Conn, error) {
	// if the pool is shutdown behave has if it
	// is simply handing out null connections
	pool.shutdownMtx.Lock()
	if pool.shutdown {
		defer pool.shutdownMtx.Unlock()
		return nil, nil
	}

	var err error

	// get from pool
	con := <-pool.cons
	pool.shutdownMtx.Unlock()

	// create new instance if none were available
	if con == nil {
		con, err = pool.dialer.Dial(pool.network, pool.address)
		if err != nil {
			pool.shutdownMtx.Lock()
			if !pool.shutdown && cap(pool.cons) > len(pool.cons) {
				pool.cons <- nil
			}
			pool.shutdownMtx.Unlock()
			return nil, err
		}

		// to tcp-tls if needed
		if pool.protocol == "tcp-tls" {
			con = tls.Client(con, &pool.tlsConfig)
		}
	} else {
		log.Tracef("Reusing connection %s", pool.address)
	}

	// return the pooled connection
	return &dns.Conn{
		Conn: con,
	}, nil
}

func (pool dnsPool) Release(conn *dns.Conn) {
	// if the pool is shutdown then close the connection
	pool.shutdownMtx.Lock()
	if pool.shutdown {
		defer pool.shutdownMtx.Unlock()
		conn.Close()
		return
	}
	pool.shutdownMtx.Unlock()

	// udp connections are not reused
	if pool.network == "udp" {
		conn.Close()
		conn.Conn = nil
	}

	// probably shutdown, don't return to a full pool chan
	pool.shutdownMtx.Lock()
	if pool.shutdown || cap(pool.cons) == len(pool.cons) {
		defer pool.shutdownMtx.Unlock()
		return
	}
	pool.shutdownMtx.Unlock()

	pool.cons <- conn.Conn
}

func (pool dnsPool) Discard(conn *dns.Conn) {
	// if the pool is not shutdown...
	pool.shutdownMtx.Lock()
	if !pool.shutdown && cap(pool.cons) != len(pool.cons) {
		// provide a nil instance back to the channel
		pool.cons <- nil
	}
	pool.shutdownMtx.Unlock()

	if conn != nil {
		// close the connection that will not be reused
		conn.Close()
		log.Debugf("Discarding connection %s", conn.Conn.RemoteAddr())
	} else {
		log.Debugf("Discarding nil connection")
	}
}

func (pool dnsPool) Shutdown() {
	pool.shutdownMtx.Lock()
	// stop dialing / issuing connections at all
	pool.shutdown = true
	pool.shutdownMtx.Unlock()

	// fill the pool to capacity with nil instances so
	// that any waiting connections are unblocked
	for c := 0; c < cap(pool.cons)-len(pool.cons); c++ {
		pool.cons <- nil
	}

	// close pool, mutex should cleanly handle it so that
	// no further writes are made to the pool
	close(pool.cons)

	log.Debugf("DnsPool %s shutdown", pool.address)
}
