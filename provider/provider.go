package provider

import (
	"fmt"
	"net"
	"os"
	"strings"

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

func (provider *provider) serve(netType string, addr string, sChan chan *dns.Server) {
	server := &dns.Server{Addr: addr, Net: netType}
	sChan <- server
	log.Infof("Listen to %s on address: %s", strings.ToUpper(netType), addr)
	if err := server.ListenAndServe(); err != nil {
		log.Errorf("Failed starting %s server: %s", netType, err.Error())
		return
	}
}

func (provider *provider) listen(listener net.Listener, packetConn net.PacketConn, sChan chan *dns.Server) {
	server := &dns.Server{}
	sChan <- server
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

	if err := server.ActivateAndServe(); err != nil {
		log.Errorf("Failed to listen: %s", err.Error())
		return
	}
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
	}

	// write response to response writer
	writer.WriteMsg(response)

	// we were having some errors during write that we need to figure out
	// and this is a good(??) way to try and find them out.
	if recovery := recover(); recovery != nil {
		log.Errorf("recovered from error: %s", recovery)
	}

	if response != nil {
		// write to query log
		if provider.qlog != nil {
			provider.qlog.Log(address, request, response, rCon, result)
		}

		// write metrics
		if provider.metrics != nil {
			provider.metrics.RecordQueryMetrics(request, response, rCon, result)
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

	// create function to handle collecting servers
	sChan := make(chan *dns.Server)
	go func() {
		for {
			select {
			case server := <-sChan:
				if server == nil {
					break
				}
				provider.servers = append(provider.servers, server)
			}
		}
		// done with channel
		close(sChan)
	}()

	// open interface connections
	if *netConf.Systemd && len(fileSockets) > 0 {
		log.Infof("Using [%d] systemd listeners...", len(fileSockets))
		for _, f := range fileSockets {
			// check if udp
			if pc, err := net.FilePacketConn(f); err == nil {
				go provider.listen(nil, pc, sChan)
				f.Close()
			} else if pc, err := net.FileListener(f); err == nil { // then check if tcp
				go provider.listen(pc, nil, sChan)
				f.Close()
			}
		}
	}

	if len(interfaces) > 0 {
		log.Infof("Using [%d] configured interfaces...", len(interfaces))
		for _, iface := range interfaces {
			addr := fmt.Sprintf("%s:%d", iface.IP, iface.Port)
			if *iface.TCP {
				go provider.serve("tcp", addr, sChan)
			}
			if *iface.UDP {
				go provider.serve("udp", addr, sChan)
			}
		}
	}

	// send nil to channel to close it
	sChan <- nil

	return nil
}

func (provider *provider) Shutdown() error {

	for _, server := range provider.servers {
		if server != nil {
			err := server.Shutdown()
			if err != nil {
				log.Errorf("During shutdown: %s", err)
			} else {
				log.Infof("Shtudown server: %s", server.Addr)
			}
		}
	}
	return nil
}
