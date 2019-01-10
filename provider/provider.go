package provider

import (
	"fmt"
	"strconv"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
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

func serve(netType string, host string, port int) {
	addr := host + ":" + strconv.Itoa(port)
	fmt.Printf("Starting %s server on address: %s ...\n", netType, addr)
	server := &dns.Server{Addr: addr, Net: netType, TsigSecret: nil}
	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Failed to setup the %s server: %s\n", netType, err.Error())
	}
}

func (provider *provider) Host(config *config.GudgeonConfig, engine engine.Engine) error {
	// get network config
	netConf := config.Network
	if netConf == nil {
		// todo: log no network structure
		return nil
	}

	// interfaces
	interfaces := netConf.Interfaces
	if interfaces == nil || len(interfaces) < 1 {
		// todo: log no interfaces
		return nil
	}

	// if no engine provided return nil
	if engine != nil {
		provider.engine = engine
	}

	// global dns handle function
	dns.HandleFunc(".", func(writer dns.ResponseWriter, request *dns.Msg) {
		if provider.engine != nil {
			engine.Handle(writer, request)
		} else {
			// when no engine defined return that there was a server failure
			response := new(dns.Msg)
			response.SetReply(request)
			response.Rcode = dns.RcodeServerFailure
			writer.WriteMsg(response)
		}
	})

	defaultTcp := true
	defaultUdp := true
	for _, iface := range interfaces {
		if defaultTcp {
			go serve("tcp", iface.IP, iface.Port)
		}
		if defaultUdp {
			go serve("udp", iface.IP, iface.Port)
		}
	}

	return nil
}
