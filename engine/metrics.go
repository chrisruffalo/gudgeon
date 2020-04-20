package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
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
	QueryTimeAvg           = "query-time-avg"
	// cache entries
	CurrentCacheEntries = "cache-entries"
	// runtime metrics
	GoRoutines         = "goroutines"
	Threads            = "process-threads"
	CurrentlyAllocated = "allocated-bytes"    // heap allocation in go runtime stats
	UsedMemory         = "process-used-bytes" // from the process api
	// cpu metrics
	CPUHundredsPercent = "cpu-hundreds-percent" // 17 == 0.17 percent, expressed in integer terms
	// worker metrics (should be qualified with -workertype, ie: "gudgeon-workers-tcp")
	Workers = "workers"
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
	JsonBytes       []byte             `json:"-"`
	Values          map[string]*Metric `json:"Values,omitempty"`
	IntervalSeconds int
}

type Metric struct {
	Count int64 `json:"count"`
}

func (metric *Metric) Set(newValue int64) *Metric {
	metric.Count = newValue
	return metric
}

func (metric *Metric) Inc(byValue int64) *Metric {
	metric.Count = metric.Count + byValue
	return metric
}

func (metric *Metric) Clear() *Metric {
	metric.Set(0)
	return metric
}

func (metric *Metric) Value() int64 {
	return metric.Count
}

type metrics struct {
	// keep config
	config   *config.GudgeonConfig
	interval *time.Duration
	duration *time.Duration

	// metrics update single allocations
	pid     int
	proc    *process.Process
	memStat *runtime.MemStats

	metricsMap   map[string]*Metric
	metricsMutex sync.RWMutex

	metricsInfoChan chan *metricsInfo
	db              *sql.DB

	cacheSizeFunc CacheSizeFunction

	// time management for interval insert
	lastInsert time.Time
	ticker     *time.Ticker

	// pool
	entryPool sync.Pool

	// metrics query cache
	queryCache *cache.Cache
}

type CacheSizeFunction = func() int64

// allows the same query and row scan logic to share code
type MetricsAccumulator = func(entry *MetricsEntry)

// query options
type QueryOptions struct {
	// a list of metrics names to "choose"
	// to return
	ChosenMetrics string
	// will be used as a modulus to skip rows
	// which reduces the resolution of a given
	// metrics stream and makes the graph
	// less cluttered
	StepSize int
}

type Metrics interface {
	GetAll() *map[string]*Metric
	Get(name string) *Metric

	// duration of metrics retention
	Duration() *time.Duration

	// duration between metrics collection
	Interval() *time.Duration

	// use cache function
	UseCacheSizeFunction(function CacheSizeFunction)

	// Query metrics from db
	Query(start time.Time, end time.Time) ([]*MetricsEntry, error)
	QueryFunc(accumulatorFunction MetricsAccumulator, options QueryOptions, unmarshall bool, start time.Time, end time.Time) error

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

func NewMetrics(config *config.GudgeonConfig, db *sql.DB) Metrics {
	metrics := &metrics{
		config:     config,
		metricsMap: make(map[string]*Metric),
		pid:        os.Getpid(),
		memStat:    &runtime.MemStats{},
	}
	metrics.proc, _ = process.NewProcess(int32(metrics.pid))

	if duration, err := util.ParseDuration(config.Metrics.Duration); err == nil {
		metrics.duration = &duration
	} else {
		log.Errorf("Error parsing metric duration: %s", err)
	}

	if interval, err := util.ParseDuration(config.Metrics.Interval); err == nil {
		metrics.interval = &interval
	} else {
		log.Errorf("Error parsing metric interval: %s", err)
	}

	if *(config.Metrics.Persist) {
		// use provided/shared db
		metrics.db = db

		// init lifetime metric counts
		metrics.load()
	}

	// update metrics initially
	metrics.update()

	// start metrics insert worker
	metrics.lastInsert = time.Now()

	// create entry pool
	metrics.entryPool = sync.Pool{
		New: func() interface{} {
			return &MetricsEntry{}
		},
	}

	// create stmt cache
	metrics.queryCache = cache.New(time.Minute*5, time.Minute)
	// and evict items on close
	metrics.queryCache.OnEvicted(func(s string, i interface{}) {
		if stmt, ok := i.(*sql.Stmt); ok {
			err := stmt.Close()
			if err != nil {
				log.Errorf("During close/evict: %s", err)
			}
		}
	})

	return metrics
}

func (metrics *metrics) GetAll() *map[string]*Metric {
	return &metrics.metricsMap
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

func (metrics *metrics) Duration() *time.Duration {
	return metrics.duration
}

func (metrics *metrics) Interval() *time.Duration {
	return metrics.interval
}

func (metrics *metrics) update() {

	// capture queries per interval into queries per second
	if metrics.interval != nil {
		metrics.Get(QueriesPerSecond).Set(int64(math.Round(float64(metrics.Get(TotalIntervalQueries).Value()) / float64(metrics.interval.Seconds()))))
		metrics.Get(BlocksPerSecond).Set(int64(math.Round(float64(metrics.Get(BlockedIntervalQueries).Value()) / float64(metrics.interval.Seconds()))))
	}

	// capture time and divide it by total queries for that interval
	if metrics.Get(TotalIntervalQueries).Value() > 0 {
		metrics.Get(QueryTimeAvg).Set(metrics.Get(QueryTime).Value() / metrics.Get(TotalIntervalQueries).Value())
	}

	// get process
	if metrics.proc != nil {
		if percent, err := metrics.proc.CPUPercent(); err == nil {
			// right now under the assumption that processor percentage is given per cores so that
			// the maximum is 100 times the number of cores
			metrics.Get(CPUHundredsPercent).Set(int64(percent*100) / int64(runtime.NumCPU()))
		}
		if pmem, err := metrics.proc.MemoryInfo(); err == nil {
			metrics.Get(UsedMemory).Set(int64(pmem.RSS))
		}
		if threads, err := metrics.proc.NumThreads(); err == nil {
			metrics.Get(Threads).Set(int64(threads))
		}
	}

	// capture goroutines
	metrics.Get(GoRoutines).Set(int64(runtime.NumGoroutine()))

	// capture memory metrics
	runtime.ReadMemStats(metrics.memStat)
	metrics.Get(CurrentlyAllocated).Set(int64(metrics.memStat.Alloc))

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

	// increase time spent serving query
	metrics.Get(QueryTime).Inc(info.ServiceMilliseconds)

	// add cache hits
	if info.Result != nil && info.Result.Cached {
		metrics.Get(CachedQueries).Inc(1)
	}

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
	// make all metrics into a json string
	bytes, err := json.Marshal(metrics.GetAll())
	if err != nil {
		log.Errorf("Error marshalling metrics json: %s", err)
		return
	}

	stmt := "INSERT INTO metrics (FromTime, AtTime, MetricsJson, IntervalSeconds) VALUES (?, ?, ?, ?)"
	_, err = tx.Exec(stmt, metrics.lastInsert, currentTime, &bytes, int(math.Round(currentTime.Sub(metrics.lastInsert).Seconds())))
	if err != nil {
		log.Errorf("Error executing metrics statement: %s", err)
		return
	}

	// clear and restart interval
	metrics.Get(TotalIntervalQueries).Clear()
	metrics.Get(BlockedIntervalQueries).Clear()
	metrics.Get(QueryTime).Clear()
	metrics.lastInsert = currentTime
}

func (metrics *metrics) prune(tx *sql.Tx) {
	if metrics.duration != nil {
		_, err := tx.Exec("DELETE FROM metrics WHERE AtTime <= ?", time.Now().Add(-1*(*metrics.duration)))
		if err != nil {
			log.Errorf("Error pruning metrics data: %s", err)
		}
	}
}

// allows custom accumulation for either streaming or custom marshalling
func (metrics *metrics) QueryFunc(accumulatorFunction MetricsAccumulator, options QueryOptions, unmarshall bool, start time.Time, end time.Time) error {
	// don't do anything with nil accumulator
	if accumulatorFunction == nil {
		return nil
	}

	var err error
	var query *sql.Stmt

	optionsQueryKey := fmt.Sprint("%s:step=%d", options.ChosenMetrics, options.StepSize)

	// try and get query from prepared cache
	if value, found := metrics.queryCache.Get(optionsQueryKey); found {
		if q, ok := value.(*sql.Stmt); ok {
			query = q
			// re-set expiration
			metrics.queryCache.Set(optionsQueryKey, query, cache.DefaultExpiration)
		}
	}

	// if no query found then make one from scratch
	if nil == query {
		chosenMetrics := options.ChosenMetrics
		var builder strings.Builder
		builder.WriteString("SELECT FromTime, AtTime, ")
		if len(chosenMetrics) > 0 {
			first := true
			filtered := false

			keepMetrics := strings.Split(chosenMetrics, ",")
			builder.WriteString("json_set('{}', ")

			for _, value := range keepMetrics {
				// if the filter value is not in the metrics map then skip it
				if _, in := metrics.metricsMap[value]; !in {
					continue
				}
				filtered = true

				if !first {
					builder.WriteString(", ")
				}
				first = false
				builder.WriteString("'$.")
				builder.WriteString(value)
				builder.WriteString("', json_extract(MetricsJson, '$.")
				builder.WriteString(value)
				builder.WriteString("')")
			}

			// if not filtered (meaning no keepable metrics were found) then just select MetricsJson
			if !filtered {
				builder.WriteString("MetricsJson")
			} else {
				builder.WriteString(")")
			}
		} else {
			builder.WriteString("MetricsJson")
		}
		builder.WriteString(", IntervalSeconds FROM metrics WHERE FromTime >= ? AND AtTime <= ?")
		if options.StepSize > 1 {
			builder.WriteString(" AND ROWID % ? = 0")
		}
		builder.WriteString(" ORDER BY AtTime ASC")
		query, err = metrics.db.Prepare(builder.String())
		if err != nil {
			return err
		}
		metrics.queryCache.Set(optionsQueryKey, query, cache.DefaultExpiration)
	}

	// add step size as parameter for prepared query when
	// the step size is provided
	params := []interface{}{start, end}
	if options.StepSize > 1 {
		params = append(params, options.StepSize)
	}

	rows, err := query.Query(params...)
	if err != nil {
		return err
	}
	defer rows.Close()

	me := metrics.entryPool.Get().(*MetricsEntry)
	for rows.Next() {
		err = rows.Scan(&me.AtTime, &me.FromTime, &me.JsonBytes, &me.IntervalSeconds)
		if err != nil {
			log.Errorf("Error scanning for metrics query: %s", err)
			continue
		}

		// unmarshal data if requested
		if unmarshall {
			_ = util.Json.Unmarshal(me.JsonBytes, &me.Values)
		}

		// call accumulator function
		accumulatorFunction(me)
	}
	metrics.entryPool.Put(me)

	return nil
}

// traditional query that returns an array of metrics entries, good for testing, small queries
func (metrics *metrics) Query(start time.Time, end time.Time) ([]*MetricsEntry, error) {
	// no sub-second metrics
	if end.Sub(start).Seconds() < 1 {
		return []*MetricsEntry{}, nil
	}

	// this is a bit of a neat trick because we can find the size of the array by the number of intervals in it
	entries := make([]*MetricsEntry, 0, int(end.Sub(start).Seconds()/metrics.interval.Seconds())+1)
	acc := func(me *MetricsEntry) {
		if me == nil {
			return
		}
		entries = append(entries, &MetricsEntry{
			FromTime:        me.FromTime,
			AtTime:          me.AtTime,
			IntervalSeconds: me.IntervalSeconds,
			Values:          me.Values,
		})
	}
	err := metrics.QueryFunc(acc, QueryOptions{}, true, start, end)
	return entries, err
}

func (metrics *metrics) load() {
	rows, err := metrics.db.Query("SELECT MetricsJson FROM metrics ORDER BY AtTime DESC LIMIT 1")
	if err != nil {
		log.Errorf("Could not load initial metrics information: %s", err)
		return
	}
	defer rows.Close()

	var metricsJSONString []byte
	for rows.Next() {
		err = rows.Scan(&metricsJSONString)
		if err != nil {
			log.Errorf("Error scanning for metrics results: %s", err)
			continue
		}
		break
	}

	// unmarshal object
	var data map[string]*Metric
	err = util.Json.Unmarshal(metricsJSONString, &data)
	if err != nil {
		// if the database is empty this is what happens
		// if the database has unmodified data, then this _shouldn't_ happen
		log.Tracef("Error marshalling metrics data from database %s", err)
		return
	}

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
	// close prepared statements
	for _, i := range metrics.queryCache.Items() {
		if stmt, ok := i.Object.(*sql.Stmt); ok {
			err := stmt.Close()
			if err != nil {
				log.Errorf("Closing metrics query database/statement: %s", err)
			}
		}
	}

	// evict all from cache
	metrics.queryCache.Flush()
}

