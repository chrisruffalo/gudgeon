package provider

import (
	"fmt"
	"net"
	"strings"

	"github.com/coreos/go-systemd/activation"
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/resolver"
)

type provider struct {
	engine engine.Engine
	metrics metrics.Metrics
}

type Provider interface {
	Host(config *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics) error
	//UpdateConfig(config *GudgeonConfig) error
	//UpdateEngine(engine *engine.Engine) error
	//Shutdown()
}

func NewProvider() Provider {
	provider := new(provider)
	return provider
}

func serve(netType string, addr string) {
	fmt.Printf("%s on address: %s\n", strings.ToUpper(netType), addr)
	server := &dns.Server{Addr: addr, Net: netType}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed starting %s server: %s\n", netType, err.Error())
		return
	}
}

func listen(listener net.Listener, packetConn net.PacketConn) {
	server := &dns.Server{}
	if packetConn != nil {
		if t, ok := packetConn.(*net.UDPConn); ok && t != nil {
			fmt.Printf("Listen on datagram: %s\n", t.LocalAddr().String())
		} else {
			fmt.Printf("Listen on unspecified datagram\n")
		}
		server.PacketConn = packetConn
	} else if listener != nil {
		fmt.Printf("Listen on stream: %s\n", listener.Addr().String())
		server.Listener = listener
	}

	if err := server.ActivateAndServe(); err != nil {
		fmt.Printf("Failed to listen: %s\n", err.Error())
		return
	}
}

func (provider *provider) handle(writer dns.ResponseWriter, request *dns.Msg) {
	// define response
	var (
		address *net.IP
		response *dns.Msg
		rCon *resolver.RequestContext
		result *resolver.ResolutionResult
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
		fmt.Printf("recovered from error: %v\n", recovery)
	}

	// write to query log
	qlog.Log(address, request, response, rCon, result)

	// write metrics
	if provider.metrics != nil {
		provider.metrics.RecordQueryMetrics(request, response, rCon, result)
	}
}

func (provider *provider) Host(config *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics) error {
	// get network config
	netConf := config.Network
	if netConf == nil {
		// todo: log no network structure
		return nil
	}

	// file descriptors from systemd
	fileSockets := activation.Files(true)

	// interfaces
	interfaces := netConf.Interfaces

	// if no interfaces and either systemd isn't enabled or
	if (interfaces == nil || len(interfaces) < 1) && (*netConf.Systemd && len(fileSockets) < 1) {
		fmt.Printf("No interfaces provided through configuration file or systemd(enabled=%t)\n", *netConf.Systemd)
		return nil
	}

	if engine != nil {
		provider.engine = engine
	}

	if metrics != nil {
		provider.metrics = metrics
	}

	// global dns handle function
	dns.HandleFunc(".", provider.handle)

	// open interface connections
	if *netConf.Systemd && len(fileSockets) > 0 {
		fmt.Printf("Using [%d] systemd listeners...\n", len(fileSockets))
		for _, f := range fileSockets {
			// check if udp
			if pc, err := net.FilePacketConn(f); err == nil {
				go listen(nil, pc)
				f.Close()
			} else if pc, err := net.FileListener(f); err == nil { // then check if tcp
				go listen(pc, nil)
				f.Close()
			}
		}
	}

	if len(interfaces) > 0 {
		fmt.Printf("Using [%d] configured interfaces...\n", len(interfaces))
		for _, iface := range interfaces {
			addr := fmt.Sprintf("%s:%d", iface.IP, iface.Port)
			if *iface.TCP {
				go serve("tcp", addr)
			}
			if *iface.UDP {
				go serve("udp", addr)
			}
		}
	}

	return nil
}
