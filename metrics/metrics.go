package metrics

import (
	"database/sql"
	"math"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/atrox/go-migrate-rice"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/json-iterator/go"
	"github.com/miekg/dns"
	"github.com/shirou/gopsutil/process"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
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
	QueryTime              = "query-time"
	// cache entries
	CurrentCacheEntries = "cache-entries"
	// rutnime metrics
	GoRoutines         = "goroutines"
	CurrentlyAllocated = "currently-allocated-bytes"
	// cpu metrics
	CPUHundredsPercent = "cpu-hundreds-percent" // 17 == 0.17 percent, expressed in integer terms

)

type metricsInfo struct {
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
	config *config.GudgeonConfig

	metricsMap   map[string]*Metric
	metricsMutex sync.RWMutex

	metricsInfoChan chan *metricsInfo
	db              *sql.DB

	cacheSizeFunc CacheSizeFunction

	// time management for interval insert
	lastInsert time.Time
	ticker     *time.Ticker
	doneTicker chan bool
}

type CacheSizeFunction = func() int64

type Metrics interface {
	GetAll() map[string]*Metric
	Get(name string) *Metric

	// use cache function
	UseCacheSizeFunction(function CacheSizeFunction)

	// record relevant metrics based on request
	RecordQueryMetrics(request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)

	// Query metrics from db
	Query(start time.Time, end time.Time) ([]*MetricsEntry, error)

	// stop
	Stop()
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

func New(config *config.GudgeonConfig) Metrics {
	metrics := &metrics{
		config:     config,
		metricsMap: make(map[string]*Metric),
	}

	if *(config.Metrics.Persist) {
		// get path to long-standing data ({home}/'data') and make sure it exists
		dataDir := config.DataRoot()
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			os.MkdirAll(dataDir, os.ModePerm)
		}

		// open db
		dbDir := path.Join(dataDir, "metrics")
		// create directory
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			os.MkdirAll(dbDir, os.ModePerm)
		}

		dbPath := path.Join(dbDir, "metrics.db")
		db, err := sql.Open("sqlite3", dbPath + "?cache=shared&journal_mode=WAL")
		if err != nil {
			// if the file exists try removing it and opening it again
			// this could be because of change in database file formats
			// or a corrupted database
			if _, rmErr := os.Stat(dbPath); !os.IsNotExist(rmErr) {
				os.Remove(dbPath)
			}
			db, err = sql.Open("sqlite3", dbPath + "?cache=shared&journal_mode=WAL")
			if err != nil {
				return nil
			}
		}
		db.SetMaxOpenConns(1)

		// do migrations
		migrationsBox := rice.MustFindBox("metrics-migrations")

		migrationDriver, err := migraterice.WithInstance(migrationsBox)
		if err != nil {
			log.Errorf("Could not get migration instances: %s", err)
			return nil
		}

		dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			log.Errorf("Could not open db: %s", err)
			return nil
		}

		m, err := migrate.NewWithInstance("rice", migrationDriver, "sqlite3", dbDriver)
		if err != nil {
			log.Errorf("Could not migrate: %s", err)
			return nil
		}

		// migrate to best version of database
		m.Up()

		// keep store handler
		metrics.db = db

		// init lifetime metric counts
		metrics.load()

		// prune metrics after load (in case the service has been down longer than the prune interval)
		metrics.prune()
	}

	// create channel for incoming metrics and start recorder
	metrics.metricsInfoChan = make(chan *metricsInfo, 100)
	go metrics.record()

	// start ticker to persist data and update periodic metrics
	duration, _ := util.ParseDuration(config.Metrics.Interval)
	metrics.ticker = time.NewTicker(duration)
	metrics.doneTicker = make(chan bool)
	metrics.lastInsert = time.Now()

	// update metrics initially
	metrics.update()

	// start go function to monitor ticker and update metrics
	go func() {
		defer metrics.ticker.Stop()
		defer close(metrics.doneTicker)

		for {
			select {
			case <-metrics.ticker.C:
				// update periodic metrics
				metrics.update()

				// only insert/prune if a db exists
				if metrics.db != nil {
					// insert new metrics
					metrics.insert(time.Now())
					// prune old metrics
					metrics.prune()
				}
			case <-metrics.doneTicker:
				return
			}
		}
	}()

	return metrics
}

func (metrics *metrics) GetAll() map[string]*Metric {
	metrics.metricsMutex.RLock()
	defer metrics.metricsMutex.RUnlock()
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

func (metrics *metrics) update() {
	// get pid
	pid := os.Getpid()

	// get process
	process, err := process.NewProcess(int32(pid))
	if err == nil && process != nil {
		percent, err := process.CPUPercent()
		if err == nil {
			metrics.Get(CPUHundredsPercent).Set(int64(percent * 100))
		}
	}

	// capture goroutines
	metrics.Get(GoRoutines).Set(int64(runtime.NumGoroutine()))

	// capture memory metrics
	memoryStats := &runtime.MemStats{}
	runtime.ReadMemStats(memoryStats)
	metrics.Get(CurrentlyAllocated).Set(int64(memoryStats.Alloc))

	// capture cache size
	if metrics.cacheSizeFunc != nil {
		metrics.Get(CurrentCacheEntries).Set(metrics.cacheSizeFunc())
	}
}

func (metrics *metrics) record() {
	// get information from channel
	for info := range metrics.metricsInfoChan {
		// first add count to total queries
		metrics.Get(TotalQueries).Inc(1)
		metrics.Get(TotalLifetimeQueries).Inc(1)
		metrics.Get(TotalIntervalQueries).Inc(1)

		// add cache hits
		if info.result != nil && info.result.Cached {
			metrics.Get(CachedQueries).Inc(1)
		}

		// add blocked queries
		if info.result != nil && info.result.Blocked {
			metrics.Get(BlockedQueries).Inc(1)
			metrics.Get(BlockedLifetimeQueries).Inc(1)
			metrics.Get(BlockedIntervalQueries).Inc(1)

			if info.result.BlockedList != nil {
				metrics.Get("rules-blocked-" + info.result.BlockedList.ShortName()).Inc(1)
			}
		}
	}
}

func (metrics *metrics) insert(currentTime time.Time) {
	// get all metrics
	all := metrics.GetAll()

	// make into json string
	bytes, err := json.Marshal(all)
	if err != nil {
		log.Errorf("Error marshalling metrics json: %s", err)
		return
	}

	stmt := "INSERT INTO metrics (FromTime, AtTime, MetricsJson, IntervalSeconds) VALUES (?, ?, ?, ?)"
	pstmt, err := metrics.db.Prepare(stmt)
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
	metrics.lastInsert = currentTime
}

func (metrics *metrics) prune() {
	duration, _ := util.ParseDuration(metrics.config.Metrics.Duration)
	_, err := metrics.db.Exec("DELETE FROM metrics WHERE AtTime <= ?", time.Now().Add(-1*duration))
	if err != nil {
		log.Errorf("Error pruning metrics data: %s", err)
	}
}

func (metrics *metrics) Query(start time.Time, end time.Time) ([]*MetricsEntry, error) {
	rows, err := metrics.db.Query("SELECT FromTime, AtTime, MetricsJson, IntervalSeconds FROM metrics WHERE FromTime >= ? AND AtTime <= ? ORDER BY AtTime ASC", start, end)
	if err != nil {
		return []*MetricsEntry{}, err
	}
	defer rows.Close()

	results := make([]*MetricsEntry, 0)

	var (
		atTime            time.Time
		fromTime          time.Time
		metricsJsonString string
		intervalSeconds   int
	)

	for rows.Next() {
		err = rows.Scan(&atTime, &fromTime, &metricsJsonString, &intervalSeconds)
		if err != nil {
			log.Errorf("Error scanning for metrics query: %s", err)
			continue
		}
		// load entry values
		entry := &MetricsEntry{
			AtTime:          atTime,
			FromTime:        fromTime,
			IntervalSeconds: intervalSeconds,
		}
		// unmarshal string into values
		json.Unmarshal([]byte(metricsJsonString), &entry.Values)
		// add metrics to results
		results = append(results, entry)
	}

	return results, nil
}

func (metrics *metrics) load() {
	rows, err := metrics.db.Query("SELECT MetricsJson FROM metrics ORDER BY AtTime DESC LIMIT 1")
	if err != nil {
		log.Errorf("Could not load initial metrics information: %s", err)
		return
	}
	defer rows.Close()

	var metricsJsonString string
	for rows.Next() {
		err = rows.Scan(&metricsJsonString)
		if err != nil {
			log.Errorf("Error scanning for metrics results: %s", err)
			continue
		}
		if "" != metricsJsonString {
			break
		}
	}

	// can't do anything with empty string, set, or object
	metricsJsonString = strings.TrimSpace(metricsJsonString)
	if "" == metricsJsonString || "{}" == metricsJsonString || "[]" == metricsJsonString {
		return
	}

	// unmarshal object
	var data map[string]*Metric
	json.Unmarshal([]byte(metricsJsonString), &data)

	preload := []string{TotalLifetimeQueries, BlockedLifetimeQueries}
	for _, key := range preload {
		if foundMetric, found := data[MetricsPrefix+key]; found {
			metrics.Get(key).Set(foundMetric.Value())
		}
	}
}

func (metrics *metrics) UseCacheSizeFunction(function CacheSizeFunction) {
	metrics.cacheSizeFunc = function
}

func (metrics *metrics) RecordQueryMetrics(request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	msg := new(metricsInfo)
	msg.request = request
	msg.response = response
	msg.result = result
	msg.rCon = rCon
	metrics.metricsInfoChan <- msg
}

func (metrics *metrics) Stop() {
	// close db and shutdown timer if it exists
	if metrics.db != nil {
		metrics.doneTicker <- true
		metrics.insert(time.Now())
		metrics.prune()
		metrics.db.Close()
	}

	// close metrics info channel
	close(metrics.metricsInfoChan)
}
