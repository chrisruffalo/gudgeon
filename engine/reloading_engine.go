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
	handles  []*events.Handle
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
		current:  current,
		handles:  make([]*events.Handle, 0),
	}

	// establish file watch
	events.Send("file:watch:start", &events.Message{"path": confPath})
	// subscribe to topic
	handle := events.Listen("file:"+confPath, func(message *events.Message) {
		// clear all file watches
		events.Send("file:watch:clear", nil)

		// reload configuration
		conf, warnings, err := config.Load(confPath)
		if err != nil {
			log.Errorf("Could not reload engine: %s", err)
		} else {
			// print log warnings and continue
			if len(warnings) > 0 {
				for _, warn := range warnings {
					log.Warn(warn)
				}
			}

			reloading.swap(conf)

			log.Infof("Configuration updated from: '%s'", confPath)
		}

		// subscribe for new change events / ensure still subscribed
		events.Send("file:watch:start", &events.Message{"path": confPath})
	})
	reloading.handles = append(reloading.handles, handle)

	// return reloading engine
	return reloading, nil
}

// wait to swap engine until all rlocked processes have completed
// and then lock during the swap and release to resume normal operations
func (rEngine *reloadingEngine) swap(config *config.GudgeonConfig) {
	// lock engine
	rEngine.mux.Lock()
	defer rEngine.mux.Unlock()

	// shutdown old engine
	if rEngine.current != nil {
		log.Debugf("Shutting down old engine...")
		rEngine.current.Shutdown()
	}

	// build new engine
	newEngine, err := NewEngine(config)

	// if engine fails then have no engine
	if err != nil {
		log.Errorf("Could not reload engine, keeping current engine (cause: %s)", err)
		return
	}

	// use new engine after build (if no errors happened)
	rEngine.current = newEngine

	log.Debugf("Using new engine...")
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

func (engine *reloadingEngine) Close() {
	if engine.current != nil {
		engine.mux.RLock()
		engine.current.Close()
		engine.mux.RUnlock()
	}
}

func (engine *reloadingEngine) Shutdown() {
	if engine.current != nil {
		engine.mux.RLock()
		engine.current.Shutdown()
		engine.mux.RUnlock()
	}
	// since we are shutting down, close up handles
	for _, handle := range engine.handles {
		if handle != nil {
			handle.Close()
		}
	}
}

func (engine *reloadingEngine) Consumers() *[]string {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Consumers()
	}
	return nil
}

func (engine *reloadingEngine) Groups() *[]string {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Groups()
	}
	return nil
}

func (engine *reloadingEngine) Resolvers() *[]string {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Resolvers()
	}
	return nil
}

func (engine *reloadingEngine) Lists() *[]*ListEntry {
	if engine.current != nil {
		engine.mux.RLock()
		defer engine.mux.RUnlock()
		return engine.current.Lists()
	}
	return nil
}
