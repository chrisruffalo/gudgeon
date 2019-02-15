package metrics

import (
	gometrics "github.com/rcrowley/go-metrics"

	"github.com/miekg/dns"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
)

const (
	// metrics prefix
	MetricsPrefix = "gudgeon-"
	// metrics names are prefixed by the metrics prefix and delim
	TotalRules     = "total-rules"
	TotalQueries   = "total-queries"
	CachedQueries  = "cached-queries"
	BlockedQueries = "blocked-queries"
	QueryTime      = "query-time"
)

type metricsInfo struct {
	request  *dns.Msg
	response *dns.Msg
	result   *resolver.ResolutionResult
	rCon     *resolver.RequestContext
}

type metrics struct {
	registry        gometrics.Registry
	metricsInfoChan chan *metricsInfo
}

type Metrics interface {
	GetAll() map[string]map[string]interface{}
	GetMeter(name string) gometrics.Meter
	GetGauge(name string) gometrics.Gauge
	GetCounter(name string) gometrics.Counter
	GetTimer(name string) gometrics.Timer

	// record relevant metrics based on request
	RecordQueryMetrics(request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)
}

func New(config *config.GudgeonConfig) Metrics {
	metrics := &metrics{}
	metrics.registry = gometrics.NewPrefixedRegistry(MetricsPrefix)

	// create metrics things that we want to be ready to at the first query every time
	gometrics.GetOrRegisterMeter(TotalQueries, metrics.registry)
	gometrics.GetOrRegisterCounter(TotalRules, metrics.registry)
	gometrics.GetOrRegisterMeter(CachedQueries, metrics.registry)
	gometrics.GetOrRegisterMeter(BlockedQueries, metrics.registry)
	gometrics.GetOrRegisterTimer(QueryTime, metrics.registry)

	// create channel and start recorder
	metrics.metricsInfoChan = make(chan *metricsInfo, 100)
	go metrics.record()

	return metrics
}

func (metrics *metrics) GetMeter(name string) gometrics.Meter {
	return gometrics.GetOrRegisterMeter(name, metrics.registry)
}

func (metrics *metrics) GetGauge(name string) gometrics.Gauge {
	return gometrics.GetOrRegisterGauge(name, metrics.registry)
}

func (metrics *metrics) GetCounter(name string) gometrics.Counter {
	return gometrics.GetOrRegisterCounter(name, metrics.registry)
}

func (metrics *metrics) GetTimer(name string) gometrics.Timer {
	return gometrics.GetOrRegisterTimer(name, metrics.registry)
}

func (metrics *metrics) GetAll() map[string]map[string]interface{} {
	return metrics.registry.GetAll()
}

func (metrics *metrics) record() {
	// get information from channel
	for info := range metrics.metricsInfoChan {
		// first add count to total queries
		queryMeter := metrics.GetMeter(TotalQueries)
		queryMeter.Mark(1)

		// add cache hits
		if info.result != nil && info.result.Cached {
			cachedMeter := metrics.GetMeter(CachedQueries)
			cachedMeter.Mark(1)
		}

		// add blocked queries
		if info.result != nil && info.result.Blocked {
			blockedMeter := metrics.GetMeter(BlockedQueries)
			blockedMeter.Mark(1)
		}
	}
}

func (metrics *metrics) RecordQueryMetrics(request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	msg := new(metricsInfo)
	msg.request = request
	msg.response = response
	msg.result = result
	msg.rCon = rCon
	metrics.metricsInfoChan <- msg
}
