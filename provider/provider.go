package provider

import (
	"fmt"
	"net"
	"strconv"

	"github.com/coreos/go-systemd/activation"
	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/qlog"
)

type provider struct {
	engine engine.Engine
}

type Provider interface {
	Host(config *config.GudgeonConfig, engine engine.Engine) error
	//UpdateConfig(config *GudgeonConfig) error
	//UpdateEngine(engine *engine.Engine) error
	//Shutdown()
}

func NewProvider() Provider {
	provider := new(provider)
	return provider
}

func serve(netType string, addr string) {
	fmt.Printf("Server %s on address: %s ...\n", netType, addr)
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
	var response *dns.Msg

	// if an engine is available actually provide some resolution
	if provider.engine != nil {
		// make query and get information back for metrics/logging
		address, eResponse, rCon, result := provider.engine.Handle(writer, request)
		response = eResponse
		// goroutine log which is async on the other side
		qlog.Log(address, request, response, rCon, result)
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
}

func (provider *provider) Host(config *config.GudgeonConfig, engine engine.Engine) error {
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
	if (interfaces == nil || len(interfaces) < 1) && (netConf.Systemd && len(fileSockets) < 1) {
		fmt.Printf("No interfaces provided through configuration file or systemd(enabled=%t)\n", netConf.Systemd)
		return nil
	}

	// if no engine provided return nil
	if engine != nil {
		provider.engine = engine
	}

	// global dns handle function
	dns.HandleFunc(".", provider.handle)

	defaultTcp := true
	defaultUdp := true

	if netConf.Systemd && len(fileSockets) > 0 {
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
			addr := iface.IP + ":" + strconv.Itoa(iface.Port)
			if defaultTcp {
				go serve("tcp", addr)
			}
			if defaultUdp {
				go serve("udp", addr)
			}
		}
	}

	return nil
}
