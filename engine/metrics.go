package engine

import (
	"database/sql"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	// metrics prefix
	MetricsPrefix = "gudgeon-"
	// metrics names are prefixed by the metrics prefix and delim
	TotalRules             = "active-rules"
	TotalQueries           = "total-session-queries"
	TotalLifetimeQueries   = "total-lifetime-queries"
	TotalIntervalQueries   = "total-interval-queries"
	CachedQueries          = "cached-queries"
	BlockedQueries         = "blocked-session-queries"
	BlockedLifetimeQueries = "blocked-lifetime-queries"
	BlockedIntervalQueries = "blocked-interval-queries"
	QueriesPerSecond       = "session-queries-ps"
	BlocksPerSecond        = "session-blocks-ps"
	QueryTime              = "query-time"
	// cache entries
	CurrentCacheEntries = "cache-entries"
	// runtime metrics
	GoRoutines         = "goroutines"
	Threads            = "process-threads"
	CurrentlyAllocated = "allocated-bytes"    // heap allocation in go runtime stats
	UsedMemory         = "process-used-bytes" // from the process api
	FreeMemory         = "free-memory-bytes"
	SystemMemory       = "system-memory-bytes"
	// cpu metrics
	CPUHundredsPercent = "cpu-hundreds-percent" // 17 == 0.17 percent, expressed in integer terms
)

type metricsInfo struct {
	address  string
	request  *dns.Msg
	response *dns.Msg
	result   *resolver.ResolutionResult
	rCon     *resolver.RequestContext
}

type MetricsEntry struct {
	FromTime        time.Time
	AtTime          time.Time
	Values          map[string]*Metric
	IntervalSeconds int
}

type  Metric struct {
	Avg int64 `json:"average,omitempty"`
	Records int64 `json:"records,omitempty"`
	Count int64 `json:"count"`
}

func (metric *Metric) Set(newValue int64) *Metric {
	metric.Count = newValue
	return metric
}

func (metric *Metric) Inc(value int64) *Metric {
	metric.Count = metric.Count + value
	return metric
}

func (metric *Metric) RecordSample(value int64) *Metric {
	metric.Count = metric.Count + value
	metric.Records = metric.Records + 1
	metric.Avg = metric.Count / metric.Records
	return metric
}

func (metric *Metric) Clear() *Metric {
	metric.Set(0)
	metric.Records = 0
	metric.Avg = 0
	return metric
}

func (metric *Metric) Value() int64 {
	return metric.Count
}

func (metric *Metric) Average() int64 {
	return metric.Avg
}

type metrics struct {
	// keep config
	config *config.GudgeonConfig

	metricsMap   map[string]*Metric
	metricsMutex sync.RWMutex

	metricsInfoChan chan *metricsInfo
	db              *sql.DB

	// metrics keep duration
	duration *time.Duration

	cacheSizeFunc CacheSizeFunction

	// time management for interval insert
	lastInsert time.Time
	ticker     *time.Ticker
}

type CacheSizeFunction = func() int64

type Metrics interface {
	GetAll() map[string]*Metric
	Get(name string) *Metric

	// use cache function
	UseCacheSizeFunction(function CacheSizeFunction)

	// Query metrics from db
	Query(start time.Time, end time.Time) ([]*MetricsEntry, error)
	QueryStream(returnChan chan *MetricsEntry, start time.Time, end time.Time) error

	// top information
	TopClients(limit int) []*TopInfo
	TopDomains(limit int) []*TopInfo
	TopQueryTypes(limit int) []*TopInfo
	TopLists(limit int) []*TopInfo
	TopRules(limit int) []*TopInfo

	// stop the metrics collection
	Stop()

	// package db management methods
	update()
	insert(tx *sql.Tx, currentTime time.Time)
	record(info *InfoRecord)
	flush(tx *sql.Tx)
	prune(tx *sql.Tx)
}

// write all metrics out to encoder
var json = jsoniter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true,
	ObjectFieldMustBeSimpleString: true,
	SortMapKeys:                   false,
	ValidateJsonRawMessage:        true,
	DisallowUnknownFields:         false,
}.Froze()

func NewMetrics(config *config.GudgeonConfig, db *sql.DB) Metrics {
	metrics := &metrics{
		config:     config,
		metricsMap: make(map[string]*Metric),
	}

	if *(config.Metrics.Persist) {
		// use provided/shared db
		metrics.db = db

		// init lifetime metric counts
		metrics.load()
	}

	// parse duration once
	if "" != config.Metrics.Duration {
		duration, err := util.ParseDuration(config.Metrics.Duration)
		if err != nil {
			metrics.duration = &duration
		}
	}

	// update metrics initially
	metrics.update()

	// start metrics insert worker
	metrics.lastInsert = time.Now()

	return metrics
}

func (metrics *metrics) GetAll() map[string]*Metric {
	// wish i could just wrap this in an immutable map
	return metrics.metricsMap
}

func (metrics *metrics) Get(name string) *Metric {
	metrics.metricsMutex.RLock()
	if metric, found := metrics.metricsMap[MetricsPrefix+name]; found {
		defer metrics.metricsMutex.RUnlock()
		return metric
	}
	metrics.metricsMutex.RUnlock()
	metrics.metricsMutex.Lock()
	defer metrics.metricsMutex.Unlock()

	metric := &Metric{Count: 0}
	metrics.metricsMap[MetricsPrefix+name] = metric
	return metric
}

// capture memory metrics
var proc, _ = process.NewProcess(int32(os.Getpid()))
var memoryStats = &runtime.MemStats{}
func (metrics *metrics) update() {

	// capture queries per interval into queries per second

	if metrics.duration != nil {
		metrics.Get(QueriesPerSecond).Set(int64(math.Round(float64(metrics.Get(TotalIntervalQueries).Value()) / float64(metrics.duration.Seconds()))))
		metrics.Get(BlocksPerSecond).Set(int64(math.Round(float64(metrics.Get(BlockedIntervalQueries).Value()) / float64(metrics.duration.Seconds()))))
	}

	// get process
	if proc != nil {
		if percent, err := proc.CPUPercent(); err == nil {
			metrics.Get(CPUHundredsPercent).Set(int64(percent * 100))
		}
		if pmem, err := proc.MemoryInfo(); err == nil {
			metrics.Get(UsedMemory).Set(int64(pmem.RSS))
		}
		if threads, err := proc.NumThreads(); err == nil {
			metrics.Get(Threads).Set(int64(threads))
		}
		if vmem, err := mem.VirtualMemory(); err == nil {
			metrics.Get(FreeMemory).Set(int64(vmem.Free))
			metrics.Get(SystemMemory).Set(int64(vmem.Total))
		}
	}

	// capture goroutines
	metrics.Get(GoRoutines).Set(int64(runtime.NumGoroutine()))

	runtime.ReadMemStats(memoryStats)
	metrics.Get(CurrentlyAllocated).Set(int64(memoryStats.Alloc))

	// capture cache size
	if metrics.cacheSizeFunc != nil {
		metrics.Get(CurrentCacheEntries).Set(metrics.cacheSizeFunc())
	}
}

func (metrics *metrics) record(info *InfoRecord) {
	// first add count to total queries
	metrics.Get(TotalQueries).Inc(1)
	metrics.Get(TotalLifetimeQueries).Inc(1)
	metrics.Get(TotalIntervalQueries).Inc(1)

	// add cache hits
	if info.Result != nil && info.Result.Cached {
		metrics.Get(CachedQueries).Inc(1)
	}

	// add latency
	metrics.Get(QueryTime).RecordSample(info.ServiceMilliseconds)

	// add blocked queries
	if info.Result != nil && (info.Result.Blocked || info.Result.Match == rule.MatchBlock) {
		metrics.Get(BlockedQueries).Inc(1)
		metrics.Get(BlockedLifetimeQueries).Inc(1)
		metrics.Get(BlockedIntervalQueries).Inc(1)

		if info.Result.MatchList != nil {
			metrics.Get("rules-session-matched-" + info.Result.MatchList.ShortName()).Inc(1)
			metrics.Get("rules-lifetime-matched-" + info.Result.MatchList.ShortName()).Inc(1)
		}
	}
}

func (metrics *metrics) insert(tx *sql.Tx, currentTime time.Time) {
	// get all metrics
	all := metrics.GetAll()

	// make into json string
	bytes, err := json.Marshal(all)
	if err != nil {
		log.Errorf("Error marshalling metrics json: %s", err)
		return
	}

	stmt := "INSERT INTO metrics (FromTime, AtTime, MetricsJson, IntervalSeconds) VALUES (?, ?, ?, ?)"
	pstmt, err := tx.Prepare(stmt)
	if err != nil {
		log.Errorf("Error preparing metrics statement: %s", err)
		return
	}
	defer pstmt.Close()

	_, err = pstmt.Exec(metrics.lastInsert, currentTime, string(bytes), int(math.Round(currentTime.Sub(metrics.lastInsert).Seconds())))
	if err != nil {
		log.Errorf("Error executing metrics statement: %s", err)
		return
	}

	// clear and restart interval
	metrics.Get(TotalIntervalQueries).Clear()
	metrics.Get(BlockedIntervalQueries).Clear()
	//metrics.Get(QueriesPerSecond).Clear()
	//metrics.Get(BlocksPerSecond).Clear()
	metrics.lastInsert = currentTime
}

func (metrics *metrics) prune(tx *sql.Tx) {
	duration, _ := util.ParseDuration(metrics.config.Metrics.Duration)
	_, err := tx.Exec("DELETE FROM metrics WHERE AtTime <= ?", time.Now().Add(-1*duration))
	if err != nil {
		log.Errorf("Error pruning metrics data: %s", err)
	}
}

// allows the same query and row scan logic to share code
type metricsAccumulator = func(entry *MetricsEntry)

// implementation of the underlying query function
func (metrics *metrics) query(qA metricsAccumulator, start time.Time, end time.Time) error {
	// don't do anything with nil accumulator
	if qA == nil {
		return nil
	}

	rows, err := metrics.db.Query("SELECT FromTime, AtTime, MetricsJson, IntervalSeconds FROM metrics WHERE FromTime >= ? AND AtTime <= ? ORDER BY AtTime ASC", start, end)
	if err != nil {
		return err
	}
	defer rows.Close()

	var (
		metricsJSONString string
	)

	var me *MetricsEntry
	for rows.Next() {
		me = &MetricsEntry{}

		err = rows.Scan(&me.AtTime, &me.FromTime, &metricsJSONString, &me.IntervalSeconds)
		if err != nil {
			log.Errorf("Error scanning for metrics query: %s", err)
			continue
		}
		// unmarshal string into values
		json.Unmarshal([]byte(metricsJSONString), &me.Values)

		// call accumulator function
		qA(me)
	}

	return nil
}

// traditional query that returns an arry of metrics entries, good for testing, small queries
func (metrics *metrics) Query(start time.Time, end time.Time) ([]*MetricsEntry, error) {
	entries := make([]*MetricsEntry, 0, 100)
	acc := func(me *MetricsEntry) {
		if me == nil {
			return
		}
		entries = append(entries, me)
	}
	err := metrics.query(acc, start, end)
	return entries, err
}

// less traditional query type that allows the web endpoint to stream the json back out as rows are scanned
func (metrics *metrics) QueryStream(returnChan chan *MetricsEntry, start time.Time, end time.Time) error {
	acc := func(me *MetricsEntry) {
		if me == nil {
			return
		}
		returnChan <- me
	}
	err := metrics.query(acc, start, end)
	close(returnChan)
	return err
}

func (metrics *metrics) load() {
	rows, err := metrics.db.Query("SELECT MetricsJson FROM metrics ORDER BY AtTime DESC LIMIT 1")
	if err != nil {
		log.Errorf("Could not load initial metrics information: %s", err)
		return
	}
	defer rows.Close()

	var metricsJSONString string
	for rows.Next() {
		err = rows.Scan(&metricsJSONString)
		if err != nil {
			log.Errorf("Error scanning for metrics results: %s", err)
			continue
		}
		if "" != metricsJSONString {
			break
		}
	}

	// can't do anything with empty string, set, or object
	metricsJSONString = strings.TrimSpace(metricsJSONString)
	if "" == metricsJSONString || "{}" == metricsJSONString || "[]" == metricsJSONString {
		return
	}

	// unmarshal object
	var data map[string]*Metric
	json.Unmarshal([]byte(metricsJSONString), &data)

	// load any metric that has "lifetime" in the key
	// from the database so that we can manage rules
	// this way as well
	for key, metric := range data {
		if strings.Contains(key, "lifetime") {
			metrics.Get(key[len(MetricsPrefix):]).Set(metric.Value())
		}
	}
}

func (metrics *metrics) UseCacheSizeFunction(function CacheSizeFunction) {
	metrics.cacheSizeFunc = function
}

func (metrics *metrics) Stop() {

}
