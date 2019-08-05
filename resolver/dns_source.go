package resolver

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/util"
)

const (
	defaultPort    = uint(53)
	defaultTLSPort = uint(853)
	portDelimeter  = ":"
	protoDelimeter = "/"
)

// how many workers to spawn
const workers = 2

// how many requests to buffer
const requestBuffer = 10

// how long to wait before source is active again
var backoffInterval = 500 * time.Millisecond

// how long to wait before timing out the connection
var defaultDeadline = 1 * time.Second

var validProtocols = []string{"udp", "tcp", "tcp-tls"}

type dnsWork struct {
	message      *dns.Msg
	responseChan chan *dnsWorkResponse
}

type dnsWorkResponse struct {
	err      error
	response *dns.Msg
}

type dnsSource struct {
	dnsServer     string
	port          uint
	remoteAddress string
	protocol      string
	network       string

	dialer net.Dialer

	backoffTime *time.Time
	tlsConfig   *tls.Config

	questionChan chan *dnsWork
	closeChan    chan bool

	workerGroup sync.WaitGroup

	sourceChanMtx sync.RWMutex
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
	dnsSource.dialer = net.Dialer{}
	// set tcp dialer properties
	if dnsSource.network == "tcp" {
		dnsSource.dialer.KeepAlive = 0
		dnsSource.dialer.Timeout = 0
	}

	// create com channels
	dnsSource.questionChan = make(chan *dnsWork, requestBuffer)
	dnsSource.closeChan = make(chan bool, workers)

	// spawn workers
	for i := 0; i < workers; i++ {
		if dnsSource.protocol == "udp" {
			go dnsSource.udpWorker()
		} else {
			go dnsSource.tcpWorker()
		}
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

func (dnsSource *dnsSource) handle(co *dns.Conn, request *dns.Msg) (*dns.Msg, error) {
	// update deadline waiting for write to succeed
	_ = co.SetDeadline(time.Now().Add(defaultDeadline))

	// write message
	if err := co.WriteMsg(request); err != nil {
		return nil, err
	}

	// read response with deadline
	_ = co.SetDeadline(time.Now().Add(2 * defaultDeadline))
	response, err := co.ReadMsg()
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (dnsSource *dnsSource) udpWorker() {
	dnsSource.workerGroup.Add(1)
	for true {
		select {
		case <-dnsSource.closeChan:
			log.Tracef("Closing '%s' udp worker", dnsSource.Name())
			dnsSource.workerGroup.Done()
			return
		case work := <-dnsSource.questionChan:
			co, err := dnsSource.connect()
			if err != nil {
				work.responseChan <- &dnsWorkResponse{err, nil}
			} else {
				response, err := dnsSource.handle(co, work.message)
				work.responseChan <- &dnsWorkResponse{err, response}
			}
		}
	}
}

func (dnsSource *dnsSource) tcpWorker() {
	// add to wait group
	dnsSource.workerGroup.Add(1)

	co, err := dnsSource.connect()
	if err != nil {
		log.Errorf("Could not establish %s connection: %s", dnsSource.protocol, err)
	}

	for true {
		select {
		case <-dnsSource.closeChan:
			if co != nil {
				err = co.Close()
				if err != nil {
					// this means something was in flight as the connection was being
					// closed and there is very little we can do at that point
					log.Debugf("Could not close connection: %s", err)
				}
			}
			log.Tracef("Closing '%s' tcp worker", dnsSource.Name())
			dnsSource.workerGroup.Done()
			return
		case work := <-dnsSource.questionChan:
			if co == nil {
				log.Tracef("opening new tcp connection in worker")
				co, err = dnsSource.connect()
				if err != nil {
					work.responseChan <- &dnsWorkResponse{err, nil}
					if co != nil {
						_ = co.Close()
					}
					co = nil
				}
			}
			if co != nil {
				response, err := dnsSource.handle(co, work.message)
				// reopen connection on error
				if err != nil {
					_ = co.Close()
					co = nil
					// if eof or broken pipe it probably just means we held on to the connection too long
					// and we can just reopen it and try again
					if nErr, ok := err.(*net.OpError); (ok && nErr.Err == syscall.EPIPE) || err == io.EOF {
						co, err = dnsSource.connect()
						if err != nil {
							// reset connection we can't make anyway and keep error for returning over channel
							co = nil
						} else {
							response, err = dnsSource.handle(co, work.message)
						}
					}
				}
				// regardless of response just return
				work.responseChan <- &dnsWorkResponse{err, response}
			}
		}
	}
}

func (dnsSource *dnsSource) query(request *dns.Msg) (*dns.Msg, error) {
	dnsSource.sourceChanMtx.RLock()
	if dnsSource.questionChan == nil {
		defer dnsSource.sourceChanMtx.RUnlock()
		return nil, fmt.Errorf("Resolver source '%s' closed", dnsSource.Name())
	}
	dnsSource.sourceChanMtx.RUnlock()

	responseChan := make(chan *dnsWorkResponse)
	dnsSource.questionChan <- &dnsWork{request, responseChan}
	answer := <-responseChan
	close(responseChan)
	return answer.response, answer.err
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
	// send enough messages to stop workers
	for i := 0; i < workers; i++ {
		dnsSource.closeChan <- true
	}

	// close input channel
	dnsSource.sourceChanMtx.Lock()
	close(dnsSource.questionChan)
	dnsSource.questionChan = nil
	dnsSource.sourceChanMtx.Unlock()

	// wait for workers to close
	dnsSource.workerGroup.Wait()
	// close response chan
	close(dnsSource.closeChan)
}
