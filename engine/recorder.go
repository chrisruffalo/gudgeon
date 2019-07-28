package engine

import (
	"database/sql"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	// might be bad to set this too high but
	// it is pretty much the only thing that
	// causes issues under load
	recordQueueSize = 100000

	// single instance of insert statement used for inserting into the "buffer"
	bufferInsertStatement = "INSERT INTO buffer (Address, ClientName, Consumer, RequestDomain, RequestType, ResponseText, Rcode, Blocked, Match, MatchList, MatchListShort, MatchRule, Cached, ServiceTime, Created, EndTime) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
)

// coordinates all recording functions/features
// and is _very tightly_ coupled to the other
// aspects
type recorder struct {
	// direct reference back to engine
	engine Engine
	conf   *config.GudgeonConfig

	// db access
	db   *sql.DB
	tx   *sql.Tx
	stmt *sql.Stmt

	// pool for creating records
	recordPool sync.Pool

	// cache lookup info
	cache     *cache.Cache
	mdnsCache *cache.Cache

	// reference to subordinate components
	qlog    QueryLog
	metrics Metrics

	// channels
	infoQueue     chan *InfoRecord
	doneChan      chan bool
	mdnsCloseChan chan bool
}

// info passed over channel and stored in database
// and that is recovered via the Query method
type InfoRecord struct {
	// client address
	Address string

	// hold the information but aren't serialized
	Request        *dns.Msg                   `json:"-"`
	Response       *dns.Msg                   `json:"-"`
	Result         *resolver.ResolutionResult `json:"-"`
	RequestContext *resolver.RequestContext   `json:"-"`

	// generated/calculated values
	Consumer       string
	ClientName     string
	ConnectionType string
	RequestDomain  string
	RequestType    string
	ResponseText   string
	Rcode          string

	// hard consumer blocked
	Blocked bool

	// matching
	Match          rule.Match
	MatchList      string
	MatchListShort string
	MatchRule      string

	// cached in resolver cache store
	Cached bool

	// when this log record was created
	Created  time.Time
	Finished time.Time

	// how long it took to service the request inside the engine
	ServiceMilliseconds int64
}

func (record *InfoRecord) clear() {
	// not set/overwritten and so need to be forced/cleared here
	record.Consumer = ""
	record.ConnectionType = ""
	record.RequestDomain = ""
	record.RequestType = ""
	record.ResponseText = ""
	record.Rcode = ""
	record.Blocked = false
	record.Match = rule.MatchNone
	record.MatchList = ""
	record.MatchListShort = ""
	record.MatchRule = ""
	record.Cached = false
	// unconditionally set when received or conditioned, no need to overwrite here
	//record.Address
	//record.Request
	//record.Response
	//record.Result
	//record.RequestContext
	//record.ClientName
	//record.Created
	//record.Finished
	//record.ServiceMilliseconds
}

func newRecord() interface{} {
	return &InfoRecord{}
}

// created from raw engine
func NewRecorder(conf *config.GudgeonConfig, engine Engine, db *sql.DB, metrics Metrics, qlog QueryLog) (*recorder, error) {
	recorder := &recorder{
		conf:      conf,
		engine:    engine,
		db:        db,
		qlog:      qlog,
		metrics:   metrics,
		infoQueue: make(chan *InfoRecord, recordQueueSize),
		doneChan:  make(chan bool),
		recordPool: sync.Pool{
			New: newRecord,
		},
	}

	// create reverse lookup cache with given ttl and given reap interval
	if *recorder.conf.QueryLog.ReverseLookup {
		recorder.mdnsCloseChan = make(chan bool)
		recorder.cache = cache.New(5*time.Minute, 10*time.Minute)
		if *recorder.conf.QueryLog.MdnsLookup {
			recorder.mdnsCache = cache.New(cache.NoExpiration, cache.NoExpiration)

			// create background channel for listening
			msgChan := make(chan *dns.Msg)
			go MulticastMdnsListen(msgChan, recorder.mdnsCloseChan)
			go CacheMulticastMessages(recorder.mdnsCache, msgChan)
		}
	}

	// if db is not nil
	if recorder.db != nil {
		// flush and prune
		recorder.flush()
		recorder.prune()
	}

	// start worker
	go recorder.worker()

	// return recorder
	return recorder, nil
}

// queue new entries, this is the method connected
// to the engine that will transfer as an async
// entry point to the worker
func (recorder *recorder) queue(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult, finishedTime *time.Time) {
	// get info record from pool
	msg := recorder.recordPool.Get().(*InfoRecord)
	msg.clear()
	msg.Address = address.String()
	msg.Request = request
	msg.Response = response
	msg.Result = result
	msg.RequestContext = rCon

	// use the start time to get started/created info
	if rCon != nil {
		msg.Created = rCon.Started
	} else {
		msg.Created = time.Now()
	}

	// use the current time or the finished time if available to get the time it was finished
	if finishedTime != nil {
		msg.Finished = *finishedTime
	} else {
		msg.Finished = time.Now()
	}

	// calculate how many milliseconds from float seconds so that we have a single-place duration
	msg.ServiceMilliseconds = int64(msg.Finished.Sub(msg.Created).Seconds() * 1000)

	// put on channel if channel is available
	if recorder.infoQueue != nil {
		recorder.infoQueue <- msg
	}
}

func (recorder *recorder) reverseLookup(info *InfoRecord) string {
	if !*recorder.conf.QueryLog.ReverseLookup {
		return ""
	}

	address := info.Address

	// look in local cache for name, even if it is empty
	if value, found := recorder.cache.Get(address); found {
		if valueString, ok := value.(string); ok {
			return valueString
		}
	}

	// look in the mdns cache
	if *recorder.conf.QueryLog.MdnsLookup && recorder.mdnsCache != nil {
		name := ReadCachedHostname(recorder.mdnsCache, address)
		if name != "" {
			return name
		}
	}

	name := ""

	// if reverse lookup is turned on query using the engine
	if *recorder.conf.QueryLog.ReverseLookup {
		name = recorder.engine.Reverse(info.Address)
		if strings.HasSuffix(name, ".") {
			name = name[:len(name)-1]
		}
	}

	// if no result from regular DNS rlookup then try and lookup the netbios name from the host
	if *recorder.conf.QueryLog.NetbiosLookup && "" == name {
		var err error
		name, err = util.LookupNetBIOSName(address)
		if err != nil {
			// don't really need to see these
			log.Tracef("During NETBIOS lookup: %s", err)
		}
	}

	if recorder.cache != nil {
		// store result, even empty results, to prevent continual lookups
		recorder.cache.Set(address, name, cache.DefaultExpiration)
	}

	return name
}

// takes the inner (request, response, context, result) information
// and moves it to relevant top-level InfoRecord information
func (recorder *recorder) condition(info *InfoRecord) {
	// condition the info item
	if info.Request != nil && len(info.Request.Question) > 0 {
		info.RequestDomain = info.Request.Question[0].Name
		info.RequestType = dns.Type(info.Request.Question[0].Qtype).String()
	}

	if info.Response != nil {
		answerValues := util.GetAnswerValues(info.Response)
		if len(answerValues) > 0 {
			info.ResponseText = answerValues[0]
		}
		info.Rcode = dns.RcodeToString[info.Response.Rcode]
	}

	if info.Result != nil {
		info.Consumer = info.Result.Consumer

		if info.Result.Blocked {
			info.Blocked = true
		}

		if info.Result.Cached {
			info.Cached = true
		}

		info.Match = info.Result.Match
		if info.Result.Match != rule.MatchNone {
			if info.Result.MatchList != nil {
				info.MatchList = info.Result.MatchList.CanonicalName()
				info.MatchListShort = info.Result.MatchList.ShortName()
			}
			info.MatchRule = info.Result.MatchRule
		}
	}

	if info.RequestContext != nil {
		info.ConnectionType = info.RequestContext.Protocol
	}

	// get reverse lookup name
	info.ClientName = recorder.reverseLookup(info)
}

// the worker is intended as the goroutine that
// acts as the switchboard for async actions so
// that only one action is performed at a time
func (recorder *recorder) worker() {
	// make timer that is only activated in some ways
	var (
		mdnsDuration   time.Duration
		mdnsQueryTimer *time.Timer
	)

	// create reverse lookup cache with given ttl and given reap interval
	if *recorder.conf.QueryLog.ReverseLookup && *recorder.conf.QueryLog.MdnsLookup {
		mdnsDuration = 1 * time.Second
		mdnsQueryTimer = time.NewTimer(mdnsDuration)
	} else {
		mdnsQueryTimer = &time.Timer{}
	}

	// start ticker to persist data and update periodic metrics
	metricsDuration, _ := util.ParseDuration(recorder.conf.Metrics.Interval)
	metricsTicker := time.NewTicker(metricsDuration)
	defer metricsTicker.Stop()
	if !(*recorder.conf.Metrics.Enabled) {
		metricsTicker.Stop()
	}

	// create ticker from conf
	duration, err := util.ParseDuration(recorder.conf.Database.Flush)
	if err != nil {
		duration = 1 * time.Second
	}
	// flush every duration
	flushTimer := time.NewTimer(duration)
	defer flushTimer.Stop()
	// prune every hour (also prunes on startup)
	pruneTimer := time.NewTimer(1 * time.Hour)
	defer pruneTimer.Stop()

	// can't do these things if there is no db
	if recorder.db == nil {
		flushTimer.Stop()
		pruneTimer.Stop()
	}

	for {
		select {
		case <-metricsTicker.C:
			// update metrics if not nil
			if recorder.metrics != nil {
				// update periodic metrics
				recorder.metrics.update()

				// only insert/prune if a db exists
				if recorder.db != nil {
					// insert new metrics inside transaction
					recorder.doWithIsolatedTransaction(func(tx *sql.Tx) {
						now := time.Now()
						recorder.metrics.insert(tx, now)
					})
				}
			}
		case info := <-recorder.infoQueue:
			// ensure record has information required
			recorder.condition(info)

			// buffer into database
			if nil != recorder.db {
				recorder.buffer(info)
			}

			// write to actual log (file or stdout)
			if recorder.qlog != nil {
				recorder.qlog.log(info)
			}

			// record metrics for single entry
			if recorder.metrics != nil {
				recorder.metrics.record(info)
			}

			// return to pool
			recorder.recordPool.Put(info)
		case <-mdnsQueryTimer.C:
			// make query
			MulticastMdnsQuery()

			// extend timer, should be exponential backoff but this is close enough
			mdnsDuration = mdnsDuration * 2
			if mdnsDuration > (30 * time.Minute) {
				mdnsDuration = 30 * time.Minute
			}
			mdnsQueryTimer.Reset(mdnsDuration)
		case <-flushTimer.C:
			log.Tracef("Flush timer triggered")
			recorder.flush()
			flushTimer.Reset(duration)
		case <-pruneTimer.C:
			log.Tracef("Prune timer triggered")
			recorder.prune()
			pruneTimer.Reset(1 * time.Hour)
		case <-recorder.doneChan:
			// when the function is over the shutdown method waits for
			// a message back on the doneChan to know that we are done
			// shutting down
			metricsTicker.Stop()
			flushTimer.Stop()
			pruneTimer.Stop()
			defer func() { recorder.doneChan <- true }()
			return
		}
	}
}

// generic method to flush transaction and then perform transaction-related function
func (recorder *recorder) doWithIsolatedTransaction(next func(tx *sql.Tx)) {
	// close any existing statements
	if recorder.stmt != nil {
		_ = recorder.stmt.Close()
		recorder.stmt = nil
	}

	if recorder.tx != nil {
		err := recorder.tx.Commit()
		recorder.tx = nil
		if err != nil {
			log.Errorf("Could not start transaction: %s", err)
			err = recorder.tx.Rollback()
			if err != nil {
				log.Errorf("Could not rollback transaction: %s", err)
			}
			return
		}
	}

	// start a new transaction for the scope of this operations
	tx, err := recorder.db.Begin()
	if err != nil {
		log.Errorf("Creating buffer flush transaction: %s", err)
		return
	}
	defer tx.Rollback()

	// do function
	next(tx)

	err = tx.Commit()
	if err != nil {
		log.Errorf("Flushing buffered entries: %s", err)
		err = tx.Rollback()
		if err != nil {
			log.Errorf("Could not rollback transaction after flush: %s", err)
		}
	}
}

// actually insert a new buffered record into the buffer
func (recorder *recorder) buffer(info *InfoRecord) {
	// only add to batch if not nil
	if info == nil {
		return
	}

	var err error

	if recorder.stmt == nil {
		recorder.stmt, err = recorder.db.Prepare(bufferInsertStatement)
		if err != nil {
			log.Errorf("Creating buffer insert prepared statement: %s", err)
		}
	}

	// insert into buffer table
	_, err = recorder.stmt.Exec(
		info.Address,
		info.ClientName,
		info.Consumer,
		info.RequestDomain,
		info.RequestType,
		info.ResponseText,
		info.Rcode,
		info.Blocked,
		info.Match,
		info.MatchList,
		info.MatchListShort,
		info.MatchRule,
		info.Cached,
		info.ServiceMilliseconds,
		info.Created,
		info.Finished,
	)
	if err != nil {
		log.Errorf("Insert into buffer: %s", err)
	}
}

// coordinate flush actions
// - close outstanding transaction
// - start transaction
// - run all subordinate flush functions
// - delete all buffered entries
// - end transaction
func (recorder *recorder) flush() {
	if recorder.db == nil {
		return
	}

	recorder.doWithIsolatedTransaction(func(tx *sql.Tx) {
		if nil != recorder.qlog {
			recorder.qlog.flush(tx)
		}

		if nil != recorder.metrics {
			recorder.metrics.flush(tx)
		}

		// empty buffer table
		_, err := tx.Exec("DELETE FROM buffer WHERE true")
		if err != nil {
			log.Errorf("Could not delete from buffer: %s", err)
		}

		// free memory
		_, err = tx.Exec(`PRAGMA shrink_memory;`)
		if err != nil {
			log.Errorf("Could not shrink memory after recorder flush: %s", err)
		}
	})
}

// coordinate prune actions
// - close outstanding transaction
// - start transaction
// - run all subordinate prune functions
// - end transaction
func (recorder *recorder) prune() {
	if recorder.db == nil {
		return
	}

	recorder.doWithIsolatedTransaction(func(tx *sql.Tx) {
		if nil != recorder.qlog {
			recorder.qlog.prune(tx)
		}

		if nil != recorder.metrics {
			recorder.metrics.prune(tx)
		}

		// free memory
		_, err := tx.Exec(`PRAGMA shrink_memory;`)
		if err != nil {
			log.Errorf("Could not shrink memory after recorder prune: %s", err)
		}
	})
}

func (recorder *recorder) shutdown() {
	// error
	var err error

	// stop accepting new entries
	infoQueue := recorder.infoQueue
	recorder.infoQueue = nil

	// signal stop mdns if enabled
	if recorder.mdnsCloseChan != nil {
		recorder.mdnsCloseChan <- true
		<-recorder.mdnsCloseChan
		close(recorder.mdnsCloseChan)
	}

	// signal done
	recorder.doneChan <- true
	<-recorder.doneChan

	// close channels
	close(infoQueue)
	close(recorder.doneChan)

	// close stmt
	if recorder.stmt != nil {
		err = recorder.stmt.Close()
		if err != nil {
			log.Errorf("Closing database/statement: %s", err)
		}
	}

	// prune and flush records
	if recorder.db != nil {
		recorder.flush()
		recorder.prune()
	}

	// stop/shutdown query log
	if nil != recorder.qlog {
		recorder.qlog.Stop()
	}

	// stop/shutdown metrics
	if nil != recorder.metrics {
		recorder.metrics.Stop()
	}

	// log shutdown
	log.Debugf("Shutdown recorder")
}
