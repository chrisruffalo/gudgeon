package resolver

import (
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	defaultPort    = uint(53)
	defaultTLSPort = uint(853)
	portDelimeter  = ":"
	protoDelimeter = "/"
)

// how long to wait before source is active again
var backoffInterval = 3 * time.Second
var defaultTimeout = 1 * time.Second

var validProtocols = []string{"udp", "tcp", "tcp-tls"}

type dnsSource struct {
	dnsServer     string
	port          uint
	remoteAddress string
	protocol      string
	network       string

	dialer net.Dialer

	backoffTime *time.Time
	tlsConfig   *tls.Config
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

	// set up tls config
	dnsSource.tlsConfig = &tls.Config{InsecureSkipVerify: true}

	// check final output
	if ip := net.ParseIP(dnsSource.dnsServer); ip != nil {
		// save/parse remote address once
		dnsSource.remoteAddress = fmt.Sprintf("%s%s%d", dnsSource.dnsServer, portDelimeter, dnsSource.port)
	}

	// keep dialer for reuse
	dnsSource.dialer = net.Dialer{
		Timeout: defaultTimeout,
	}
}

func (dnsSource *dnsSource) connect() (*dns.Conn, error) {
	conn, err := dnsSource.dialer.Dial(dnsSource.network, dnsSource.remoteAddress)
	if err != nil {
		return nil, err
	}
	if dnsSource.protocol == "tcp-tls" {
		conn = tls.Client(conn, dnsSource.tlsConfig)
	}
	return &dns.Conn{Conn: conn}, nil
}

func (dnsSource *dnsSource) query(request *dns.Msg) (*dns.Msg, error) {
	co, err := dnsSource.connect()
	if err != nil {
		return nil, err
	}
	defer co.Close()

	// update deadline waiting for write to succeed
	_ = co.SetDeadline(time.Now().Add(defaultTimeout))

	// write message
	if err := co.WriteMsg(request); err != nil {
		return nil, err
	}

	// read response with deadline
	_ = co.SetDeadline(time.Now().Add(2 * defaultTimeout))
	response, err := co.ReadMsg()
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (dnsSource *dnsSource) Answer(rCon *RequestContext, context *ResolutionContext, request *dns.Msg) (*dns.Msg, error) {
	now := time.Now()
	if dnsSource.backoffTime != nil && now.Before(*dnsSource.backoffTime) {
		// "asleep" during backoff interval
		return nil, nil
	}
	// the backoff time is irrelevant now
	dnsSource.backoffTime = nil

	// this is considered a recursive query so don't if recursion was not requested
	if request == nil || !request.MsgHdr.RecursionDesired {
		return nil, nil
	}

	// forward message without interference
	response, err := dnsSource.query(request)
	if err != nil {
		backoff := time.Now().Add(backoffInterval)
		dnsSource.backoffTime = &backoff
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

}
