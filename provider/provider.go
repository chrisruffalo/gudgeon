package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/resolver"
)

type provider struct {
	engine  engine.Engine
	metrics metrics.Metrics
	qlog    qlog.QLog
	servers []*dns.Server
}

type Provider interface {
	Host(config *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics, qlog qlog.QLog) error
	//UpdateConfig(config *GudgeonConfig) error
	//UpdateEngine(engine *engine.Engine) error
	Shutdown() error
}

func NewProvider() Provider {
	provider := new(provider)
	provider.servers = make([]*dns.Server, 0)
	return provider
}

func (provider *provider) serve(netType string, addr string) *dns.Server {
	server := &dns.Server{
		Addr:         addr,
		Net:          netType,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	log.Infof("Listen to %s on address: %s", strings.ToUpper(netType), addr)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("Failed starting %s server: %s", netType, err.Error())
		}
	}()
	return server
}

func (provider *provider) listen(listener net.Listener, packetConn net.PacketConn) *dns.Server {
	server := &dns.Server{
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	if packetConn != nil {
		if t, ok := packetConn.(*net.UDPConn); ok && t != nil {
			log.Infof("Listen on datagram: %s", t.LocalAddr().String())
		} else {
			log.Info("Listen on unspecified datagram")
		}
		server.PacketConn = packetConn
	} else if listener != nil {
		log.Infof("Listen on stream: %s", listener.Addr().String())
		server.Listener = listener
	}

	go func() {
		if err := server.ActivateAndServe(); err != nil {
			log.Errorf("Failed to listen: %s", err.Error())
		}
	}()
	return server
}

func (provider *provider) handle(writer dns.ResponseWriter, request *dns.Msg) {
	// define response
	var (
		address  *net.IP
		response *dns.Msg
		rCon     *resolver.RequestContext
		result   *resolver.ResolutionResult
	)

	// get consumer ip from request
	protocol := ""
	if ip, ok := writer.RemoteAddr().(*net.UDPAddr); ok {
		address = &(ip.IP)
		protocol = "udp"
	}
	if ip, ok := writer.RemoteAddr().(*net.TCPAddr); ok {
		address = &(ip.IP)
		protocol = "tcp"
	}

	// if an engine is available actually provide some resolution
	if provider.engine != nil {
		// make query and get information back for metrics/logging
		response, rCon, result = provider.engine.Handle(address, protocol, writer, request)
	} else {
		// when no engine defined return that there was a server failure
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeServerFailure

		// log that there is no engine to service request?
		log.Errorf("No engine to process request")
	}

	// write response to response writer
	writer.WriteMsg(response)
	// close the writer since it's done
	writer.Close()

	// we were having some errors during write that we need to figure out
	// and this is a good(??) way to try and find them out.
	if recovery := recover(); recovery != nil {
		log.Errorf("recovered from error: %s", recovery)
	}

	// only needed if a resposne was collected and the query log/metrics are enabled
	if response != nil && (provider.qlog != nil || provider.metrics != nil) {
		// write metrics
		if provider.metrics != nil {
			provider.metrics.RecordQueryMetrics(address, request, response, rCon, result)
		}

		// write to query log
		if provider.qlog != nil {
			provider.qlog.Log(address, request, response, rCon, result)
		}
	}
}

func (provider *provider) Host(config *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics, qlog qlog.QLog) error {
	// get network config
	netConf := config.Network

	// start out with no file socket descriptors
	fileSockets := []*os.File{}

	// get file descriptors from systemd and fail gracefully
	if *netConf.Systemd {
		fileSockets = activation.Files(true)
		if recovery := recover(); recovery != nil {
			log.Errorf("Could not use systemd activation: %s", recovery)
		}
	}

	// interfaces
	interfaces := netConf.Interfaces

	// if no interfaces and either systemd isn't enabled or
	if (interfaces == nil || len(interfaces) < 1) && (*netConf.Systemd && len(fileSockets) < 1) {
		log.Errorf("No interfaces provided through configuration file or systemd(enabled=%t)", *netConf.Systemd)
		return nil
	}

	if engine != nil {
		provider.engine = engine
	}

	if metrics != nil {
		provider.metrics = metrics
	}

	if qlog != nil {
		provider.qlog = qlog
	}

	// global dns handle function
	dns.HandleFunc(".", provider.handle)

	// server collector
	var server *dns.Server

	// open interface connections
	if *netConf.Systemd && len(fileSockets) > 0 {
		log.Infof("Using [%d] systemd listeners...", len(fileSockets))
		for _, f := range fileSockets {
			// check if udp
			if pc, err := net.FilePacketConn(f); err == nil {
				server = provider.listen(nil, pc)
				if server != nil {
					provider.servers = append(provider.servers, server)
					server = nil
				}
				f.Close()
			} else if pc, err := net.FileListener(f); err == nil { // then check if tcp
				server = provider.listen(pc, nil)
				if server != nil {
					provider.servers = append(provider.servers, server)
					server = nil
				}
				f.Close()
			}
		}
	}

	if len(interfaces) > 0 {
		log.Infof("Using [%d] configured interfaces...", len(interfaces))
		for _, iface := range interfaces {
			addr := fmt.Sprintf("%s:%d", iface.IP, iface.Port)
			if *iface.TCP {
				server = provider.serve("tcp", addr)
				if server != nil {
					provider.servers = append(provider.servers, server)
					server = nil
				}
			}
			if *iface.UDP {
				server = provider.serve("udp", addr)
				if server != nil {
					provider.servers = append(provider.servers, server)
					server = nil
				}
			}
		}
	}

	return nil
}

func (provider *provider) Shutdown() error {
	context, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, server := range provider.servers {
		// stop each server separately
		if server != nil {
			err := server.ShutdownContext(context)
			if err != nil {
				log.Errorf("During shutdown: %s", err)
			} else {
				log.Infof("Shtudown server: %s", server.Addr)
			}
		}
	}

	// todo: this just isn't right, maybe a waitgroup here and 
	// go routines for shutting down each server?
	<-context.Done()

	return nil
}
