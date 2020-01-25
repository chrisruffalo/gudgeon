package resolver

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/pool"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	defaultPort    = uint(53)
	defaultTLSPort = uint(853)
	portDelimeter  = ":"
	protoDelimeter = "/"
)

// how many workers to spawn
const workers = 1

// how many requests to buffer
const requestBuffer = 1

// how long to wait before timing out the connection
var defaultDeadline = 350 * time.Millisecond

var validProtocols = []string{"udp", "tcp", "tcp-tls"}


type dnsSource struct {
	dnsServer     string
	port          uint
	remoteAddress string
	protocol      string
	network       string

	pool		  pool.DnsPool
}

func (dnsSource *dnsSource) Name() string {
	return dnsSource.remoteAddress + "/" + dnsSource.protocol
}

func (dnsSource *dnsSource) Load(specification string) {
	dnsSource.port = 0
	dnsSource.dnsServer = ""
	dnsSource.protocol = ""

	// determine first if there is an attached protocol
	if strings.Contains(specification, protoDelimeter) {
		split := strings.Split(specification, protoDelimeter)
		if len(split) > 1 && util.StringIn(strings.ToLower(split[1]), validProtocols) {
			specification = split[0]
			dnsSource.protocol = strings.ToLower(split[1])
		}
	}

	// need to determine if a port comes along with the address and parse it out once
	if strings.Contains(specification, portDelimeter) {
		split := strings.Split(specification, portDelimeter)
		if len(split) > 1 {
			dnsSource.dnsServer = split[0]
			var err error
			parsePort, err := strconv.ParseUint(split[1], 10, 32)
			// recover from error
			if err != nil {
				dnsSource.port = 0
			} else {
				dnsSource.port = uint(parsePort)
			}
		}
	} else {
		dnsSource.dnsServer = specification
	}

	// set defaults if missing
	if "" == dnsSource.protocol {
		dnsSource.protocol = "udp"
	}
	// the network should be just tcp, really
	dnsSource.network = dnsSource.protocol
	if "tcp-tls" == dnsSource.protocol {
		dnsSource.network = "tcp"
	}

	// recover from parse errors or use default port in event port wasn't set
	if dnsSource.port == 0 {
		if "tcp-tls" == dnsSource.protocol {
			dnsSource.port = defaultTLSPort
		} else {
			dnsSource.port = defaultPort
		}
	}
	// check final output
	if ip := net.ParseIP(dnsSource.dnsServer); ip != nil {
		// save/parse remote address once
		dnsSource.remoteAddress = fmt.Sprintf("%s%s%d", dnsSource.dnsServer, portDelimeter, dnsSource.port)
	}

	// create pool
	dnsSource.pool = pool.DefaultDnsPool(dnsSource.protocol, dnsSource.remoteAddress)
}

func (dnsSource *dnsSource) handle(co *dns.Conn, request *dns.Msg) (*dns.Msg, error) {
	// update deadline waiting for write to succeed
	_ = co.SetWriteDeadline(time.Now().Add(defaultDeadline))

	// write message
	if err := co.WriteMsg(request); err != nil {
		return nil, err
	}

	// read response with deadline
	_ = co.SetReadDeadline(time.Now().Add(defaultDeadline))
	response, err := co.ReadMsg()

	if response != nil && response.MsgHdr.Id != request.MsgHdr.Id {
		log.Warnf("Response id (%d) does not match request id (%d) for question:\n%s", response.MsgHdr.Id, request.MsgHdr.Id, request.String())
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

func (dnsSource *dnsSource) query(request *dns.Msg) (*dns.Msg, error) {
	conn, err := dnsSource.pool.Get()
	// discard on error during connection
	if err != nil {
		dnsSource.pool.Discard(conn)
		return nil, err
	}
	// need to discard nil con to avoid jamming up the way it works
	if conn == nil {
		dnsSource.pool.Discard(conn)
		return nil, fmt.Errorf("No connection provided by pool")
	}

	response, err := dnsSource.handle(conn, request)
	if err != nil {
		dnsSource.pool.Discard(conn)
	} else {
		dnsSource.pool.Release(conn)
	}
	return response, err
}

func (dnsSource *dnsSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	// this is considered a recursive query so don't if recursion was not requested
	if request == nil || !request.MsgHdr.RecursionDesired {
		return nil, nil
	}

	// forward message without interference
	response, err := dnsSource.query(request)
	if err != nil {
		return nil, err
	}

	// do not set reply here (doesn't seem to matter, leaving this comment so nobody decides to do it in the future without cause)
	// response.SetReply(request)

	// set source as answering source
	if context != nil && !util.IsEmptyResponse(response) && context.SourceUsed == "" {
		context.SourceUsed = dnsSource.Name()
	}

	// otherwise just return
	return response, nil
}

func (dnsSource *dnsSource) Close() {
	dnsSource.pool.Shutdown()
}
