package provider

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
)

type provider struct {
	engine  engine.Engine
	servers []*dns.Server
}

type Provider interface {
	Host(config *config.GudgeonConfig, engine engine.Engine) error
	//UpdateConfig(config *GudgeonConfig) error
	//UpdateEngine(engine *engine.Engine) error
	Shutdown() error
}

func NewProvider(engine engine.Engine) Provider {
	provider := new(provider)
	provider.engine = engine
	provider.servers = make([]*dns.Server, 0)
	return provider
}

func defaultServer() *dns.Server {
	return &dns.Server{
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

func (provider *provider) serve(netType string, addr string) *dns.Server {
	server := defaultServer()
	server.Addr = addr
	server.Net = netType

	log.Infof("Listen to %s on address: %s", strings.ToUpper(netType), addr)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Errorf("Failed starting %s server: %s", netType, err.Error())
		}
	}()
	return server
}

func (provider *provider) listen(listener net.Listener, packetConn net.PacketConn) *dns.Server {
	server := defaultServer()
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
		response, _, _ = provider.engine.Handle(address, protocol, request)
	} else {
		// when no engine defined return that there was a server failure
		response = new(dns.Msg)
		response.SetReply(request)
		response.Rcode = dns.RcodeServerFailure

		// log that there is no engine to service request?
		log.Errorf("No engine to process request")
	}

	// write response to response writer
	err := writer.WriteMsg(response)
	if err != nil {
		log.Errorf("Writing response: %s", err)
	}

	// we were having some errors during write that we need to figure out
	// and this is a good(??) way to try and find them out.
	if recovery := recover(); recovery != nil {
		log.Errorf("recovered from error: %s", recovery)
	}

	// explicitly close the writer since it's done
	err = writer.Close()
	if err != nil {
		log.Errorf("Closing response: %s", err)
	}
}

func (provider *provider) Host(config *config.GudgeonConfig, engine engine.Engine) error {
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

	// global dns handle function
	dns.HandleFunc(".", provider.handle)

	// open interface connections
	if *netConf.Systemd && len(fileSockets) > 0 {
		log.Infof("Using [%d] systemd listeners...", len(fileSockets))
		for _, f := range fileSockets {
			// check if udp
			if pc, err := net.FilePacketConn(f); err == nil {
				provider.servers = append(provider.servers, provider.listen(nil, pc))
				f.Close()
			} else if pc, err := net.FileListener(f); err == nil { // then check if tcp
				provider.servers = append(provider.servers, provider.listen(pc, nil))
				f.Close()
			}
		}
	}

	if len(interfaces) > 0 {
		log.Infof("Using [%d] configured interfaces...", len(interfaces))
		for _, iface := range interfaces {

			addr := fmt.Sprintf("%s:%d", iface.IP, iface.Port)
			if *iface.TCP {
				provider.servers = append(provider.servers, provider.serve("tcp", addr))
			}
			if *iface.UDP {
				provider.servers = append(provider.servers, provider.serve("udp", addr))
			}
		}
	}

	return nil
}

func (provider *provider) Shutdown() error {
	// set with a 60 second timeout
	context, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// start a waitgroup
	wg := &sync.WaitGroup{}

	for _, server := range provider.servers {
		// stop each server separately
		if server != nil {
			// add newly started go function to wg
			wg.Add(1)
			// do a go shutdown function with wg for each server
			go func() {
				server.ShutdownContext(context)
				log.Infof("Shtudown server: %s", server.Addr)
				wg.Done()
			}()
		}
	}

	// wait for group to be done
	wg.Wait()

	return nil
}
