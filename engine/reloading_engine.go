package engine

import (
	"fmt"
	"github.com/chrisruffalo/gudgeon/events"
	"net"
	"sync"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
)

type reloadingEngine struct {
	confPath string
	current  Engine
	mux      sync.RWMutex
}

func NewReloadingEngine(confPath string, conf *config.GudgeonConfig) (Engine, error) {
	// build engine as normal
	current, err := NewEngine(conf)
	if err != nil {
		return nil, err
	}

	// create new reloading shell for engine
	reloading := &reloadingEngine{
		confPath: confPath,
		current: current,
	}

	// establish file watch
	events.Send("file:watch", &events.Message{ "path": confPath })
	// subscribe to topic
	events.Listen("file:" + confPath, func(message *events.Message) {
		// reload configuration
		config, warnings, err := config.Load(confPath)
		if err != nil {
			log.Errorf("%s", err)
		} else {
			// print log warnings and continue
			if len(warnings) > 0 {
				for _, warn := range warnings {
					log.Warn(warn)
				}
			}

			reloading.swap(config)

			log.Infof("Configuration updated from: '%s'", confPath)
		}

		// subscribe for new change events / ensure still subscribed
		events.Send("file:watch", &events.Message{ "path": confPath })
	})

	// return reloading engine
	return reloading, nil
}

// wait to swap engine until all rlocked processes have completed
// and then lock during the swap and release to resume normal operations
func (engine *reloadingEngine) swap(config *config.GudgeonConfig) {
	// lock
	engine.mux.Lock()

	// shutdown old engine
	if engine.current != nil {
		engine.current.Shutdown()
	}

	// build new engine
	newEngine, err := NewEngine(config)

	// if engine fails then have no engine
	if err != nil {
		log.Errorf("Could not reload engine: %s", err)
		engine.current = nil
		return
	}

	// use new engine
	engine.current = newEngine

	engine.mux.Unlock()
}

func (engine *reloadingEngine) IsDomainRuleMatched(consumer *net.IP, domain string) (rule.Match, *config.GudgeonList, string) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.IsDomainRuleMatched(consumer, domain)
	}
	return rule.MatchNone, nil, ""
}

func (engine *reloadingEngine) Resolve(domainName string) (string, error) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Resolve(domainName)
	}
	return "", fmt.Errorf("No engine currently available")
}

func (engine *reloadingEngine) Reverse(address string) string {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Reverse(address)
	}
	return ""
}

func (engine *reloadingEngine) Handle(address *net.IP, protocol string, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Handle(address, protocol, request)
	}
	return nil, nil, nil
}

func (engine *reloadingEngine) HandleWithConsumerName(consumerName string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.HandleWithConsumerName(consumerName, rCon, request)
	}
	return nil, nil, nil
}

func (engine *reloadingEngine) HandleWithConsumer(consumer *consumer, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.HandleWithConsumer(consumer, rCon, request)
	}
	return nil, nil, nil
}

func (engine *reloadingEngine) HandleWithGroups(groups []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.HandleWithGroups(groups, rCon, request)
	}
	return nil, nil, nil
}

func (engine *reloadingEngine) HandleWithResolvers(resolvers []string, rCon *resolver.RequestContext, request *dns.Msg) (*dns.Msg, *resolver.RequestContext, *resolver.ResolutionResult) {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.HandleWithResolvers(resolvers, rCon, request)
	}
	return nil, nil, nil
}

func (engine *reloadingEngine) CacheSize() int64 {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.CacheSize()
	}
	return int64(0)
}

func (engine *reloadingEngine) QueryLog() QueryLog {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.QueryLog()
	}
	return nil
}

func (engine *reloadingEngine) Metrics() Metrics {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Metrics()
	}
	return nil
}

func (engine *reloadingEngine) Shutdown() {
	if engine.current != nil {
		engine.mux.RLock()
		engine.current.Shutdown()
		engine.mux.RUnlock()
	}
}



