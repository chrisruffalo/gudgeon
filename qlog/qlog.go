package qlog

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/atrox/go-migrate-rice"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	"github.com/miekg/dns"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	// constant insert statement
	qlogInsertStatement = "insert into qlog (Address, ClientName, Consumer, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
)

// lit of valid sort names (lower case for ease of use with util.StringIn)
var validSorts = []string{"address", "connectiontype", "requestdomain", "requesttype", "blocked", "blockedlist", "blockedrule", "created"}

// allows a dependency injection-way of defining a reverse lookup function, takes a string address (should be an IP) and returns a string that contains the domain name result
type ReverseLookupFunction = func(addres string) string

// info passed over channel and stored in database
// and that is recovered via the Query method
type LogInfo struct {
	// client address
	Address        string

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
	Blocked        bool
	BlockedList    string
	BlockedRule    string
	Created        time.Time
}

// the type that is used to make queries against the
// query log (should be used by the web interface to
// find queries)
type QueryLogQuery struct {
	// query on fields
	Address        string
	ConnectionType string
	RequestDomain  string
	RequestType    string
	Blocked        *bool
	// query on created time
	After  *time.Time
	Before *time.Time
	// query limits for paging
	Skip  int
	Limit int
	// query sort
	SortBy  string
	Reverse *bool
}

// store database location
type qlog struct {
	rlookup     ReverseLookupFunction
	cache       *cache.Cache
	mdnsCache   *cache.Cache

	fileLogger  *log.Logger
	stdLogger   *log.Logger

	store        *sql.DB
	qlConf       *config.GudgeonQueryLog
	logInfoChan  chan *LogInfo
	doneChan     chan bool
	batch        []*LogInfo
}

// public interface
type QLog interface {
	Query(query *QueryLogQuery) []LogInfo
	Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)
	Stop()
}

func NewWithReverseLookup(conf *config.GudgeonConfig, rlookup ReverseLookupFunction) (QLog, error) {
	qlConf := conf.QueryLog
	if qlConf == nil || !*(qlConf.Enabled) {
		return nil, nil
	}

	// create new empty qlog
	qlog := &qlog{}
	qlog.qlConf = qlConf
	if qlog != nil && rlookup != nil {
		qlog.rlookup = rlookup
	}
	// create reverse lookup cache with given ttl and given reap interval
	qlog.cache = cache.New(5*time.Minute, 10*time.Minute)
	qlog.mdnsCache = cache.New(cache.NoExpiration, 10*time.Minute)

	// create distinct loggers for query output
	if qlConf.File != "" {
		// create destination and writer
		dirpart := path.Dir(qlConf.File)
		if _, err := os.Stat(dirpart); os.IsNotExist(err) {
			os.MkdirAll(dirpart, os.ModePerm)
		}

		// attempt to open file
		w, err := os.OpenFile(qlConf.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
		if err != nil {
			log.Errorf("While opening query log file: %s", err)
		} else {
			log.Infof("Logging queries to file: %s", qlConf.File)
			qlog.fileLogger = log.New()
			qlog.fileLogger.SetOutput(w)
			qlog.fileLogger.SetLevel(log.InfoLevel)
			qlog.fileLogger.SetFormatter(&log.JSONFormatter{})
		}
	}

	if *(qlConf.Stdout) {
		log.Info("Logging queries to stdout")
		qlog.stdLogger = log.New()
		qlog.stdLogger.SetOutput(os.Stdout)
		qlog.stdLogger.SetLevel(log.InfoLevel)
		qlog.stdLogger.SetFormatter(&log.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
		})
	}

	// create log channel
	qlog.batch = make([]*LogInfo, 0, qlConf.BatchSize)
	qlog.logInfoChan = make(chan *LogInfo, qlConf.BatchSize*10) // support 10 "batches" in queue
	qlog.doneChan = make(chan bool)
	go qlog.logWorker()

	// create background tasks/channels for mdns polling
	msgChan := make(chan *dns.Msg)
    go MulticastMdnsListen(msgChan)
    go CacheMulticastMessages(qlog.mdnsCache, msgChan)
    // and create exponential backoff for timer for multicast query
    go func() {
    	// create and start timer
    	duration := 1 * time.Second
    	mdnsQueryTimer := time.NewTimer(duration)

    	// wait for time and do actions
    	for _ = range mdnsQueryTimer.C {
    		// make query
    		MulticastMdnsQuery()

    		// extend timer, should be exponential backoff but this is close enough
    		duration = duration * 10
    		if duration > time.Hour {
    			duration = time.Hour
    		}
    		mdnsQueryTimer.Reset(duration)
    	}
    }()


	// only build DB if persistence is enabled
	if *(qlog.qlConf.Persist) {
		// get path to long-standing data ({home}/'data') and make sure it exists
		dataDir := conf.DataRoot()
		if _, err := os.Stat(dataDir); os.IsNotExist(err) {
			os.MkdirAll(dataDir, os.ModePerm)
		}

		// open db
		dbDir := path.Join(dataDir, "query_log")
		// create directory
		if _, err := os.Stat(dbDir); os.IsNotExist(err) {
			os.MkdirAll(dbDir, os.ModePerm)
		}

		dbPath := path.Join(dbDir, "qlog.db")
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			// if the file exists try removing it and opening it again
			// this could be because of change in database file formats
			// or a corrupted database
			if _, rmErr := os.Stat(dbPath); !os.IsNotExist(rmErr) {
				os.Remove(dbPath)

			}
			return nil, err
		}

		// do migrations
		migrationsBox := rice.MustFindBox("qlog-migrations")

		migrationDriver, err := migraterice.WithInstance(migrationsBox)
		if err != nil {
			return nil, err
		}

		dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
		if err != nil {
			return nil, err
		}

		m, err := migrate.NewWithInstance("rice", migrationDriver, "sqlite3", dbDriver)
		if err != nil {
			return nil, err
		}

		// migrate to best version of database
		m.Up()

		// keep store handler
		qlog.store = db

		// prune entries
		qlog.prune()
	}

	return qlog, nil
}

// create a new query log according to configuration
func New(conf *config.GudgeonConfig) (QLog, error) {
	return NewWithReverseLookup(conf, nil)
}

func (qlog *qlog) prune() {
	duration, _ := util.ParseDuration(qlog.qlConf.Duration)
	_, err := qlog.store.Exec("DELETE FROM qlog WHERE Created <= ?", time.Now().Add(-1*duration))
	if err != nil {
		log.Errorf("Error pruning qlog data: %s", err)
	}
}

func (qlog *qlog) logDB(info *LogInfo, forceInsert bool) {
	// only add to batch if not nil
	if info != nil {
		qlog.batch = append(qlog.batch, info)
	}

	// insert whole batch and reset batch
	if (forceInsert || len(qlog.batch) >= qlog.qlConf.BatchSize) && len(qlog.batch) > 0 {
		// attempt to insert statements
		stmt := qlogInsertStatement + strings.Repeat(", (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", len(qlog.batch)-1)
		vars := make([]interface{}, 0, len(qlog.batch)*9)
		for _, i := range qlog.batch {
			vars = append(vars, i.Address)
			vars = append(vars, i.ClientName)
			vars = append(vars, i.Consumer)
			vars = append(vars, i.RequestDomain)
			vars = append(vars, i.RequestType)
			vars = append(vars, i.ResponseText)
			vars = append(vars, i.Blocked)
			vars = append(vars, i.BlockedList)
			vars = append(vars, i.BlockedRule)
			vars = append(vars, i.Created)
		}
		pstmt, err := qlog.store.Prepare(stmt)
		defer pstmt.Close()
		if err != nil {
			log.Errorf("Error preparing statement: %s", err)
		} else {
			_, err = pstmt.Exec(vars...)
			if err != nil {
				log.Errorf("Could not insert into db: %s", err)
			}
		}

		// remake batch for inserting
		qlog.batch = make([]*LogInfo, 0, qlog.qlConf.BatchSize)
	}
}

func (qlog *qlog) log(info *LogInfo) {
	// get values
	response := info.Response
	result := info.Result
	rCon := info.RequestContext

	// create builder
	var builder strings.Builder

	var fields log.Fields
	if qlog.fileLogger != nil {
		fields = log.Fields{}
	}

	// log result if found
	builder.WriteString("[")
	if info.ClientName != "" {
		builder.WriteString(info.ClientName)
		if qlog.fileLogger != nil {
			fields["clientName"] = info.ClientName
		}
		builder.WriteString("|")
	}
	builder.WriteString(info.Address)
	builder.WriteString("/")
	builder.WriteString(rCon.Protocol)
	builder.WriteString("|")
	builder.WriteString(info.Consumer)
	builder.WriteString("] q:[")
	builder.WriteString(info.RequestDomain)
	builder.WriteString("|")
	builder.WriteString(info.RequestType)
	builder.WriteString("]->")
	if qlog.fileLogger != nil {
		fields["address"] = info.Address
		fields["protocol"] = rCon.Protocol
		fields["consumer"] = info.Consumer
		fields["requestDomain"] = info.RequestDomain
		fields["requestType"] = info.RequestType
	}

	if result != nil {
		if result.Blocked {
			builder.WriteString("BLOCKED")
			if result.BlockedList != nil {
				builder.WriteString("[")
				builder.WriteString(result.BlockedList.CanonicalName())
				if qlog.fileLogger != nil {
					fields["blockedList"] = result.BlockedList.CanonicalName()
				}
				if result.BlockedRule != "" {
					builder.WriteString("|")
					builder.WriteString(result.BlockedRule)
					if qlog.fileLogger != nil {
						fields["blockedRule"] = result.BlockedRule
					}
				}
				builder.WriteString("]")
			}
		} else {
			if result.Cached {
				builder.WriteString("c:[")
				builder.WriteString(result.Resolver)
				builder.WriteString("]")
				if qlog.fileLogger != nil {
					fields["resolver"] = result.Resolver
				}
			} else {
				builder.WriteString("r:[")
				builder.WriteString(result.Resolver)
				builder.WriteString("]")
				builder.WriteString("->")
				builder.WriteString("s:[")
				builder.WriteString(result.Source)
				builder.WriteString("]")
				if qlog.fileLogger != nil {
					fields["resolver"] = result.Resolver
					fields["source"] = result.Source
				}
			}

			builder.WriteString("->")

			if len(response.Answer) > 0 {
				answerValues := util.GetAnswerValues(response)
				if len(answerValues) > 0 {
					builder.WriteString(answerValues[0])
					if qlog.fileLogger != nil {
						fields["answer"] = answerValues[0]
					}
					if len(answerValues) > 1 {
						builder.WriteString(fmt.Sprintf(" (+%d)", len(answerValues)-1))
					}
				} else {
					builder.WriteString("(EMPTY RESPONSE)")
					if qlog.fileLogger != nil {
						fields["answer"] = "<< NONE >>"
					}
				}
			}
		}
	} else if response.Rcode == dns.RcodeServerFailure {
		// write as error and return
		if qlog.fileLogger != nil {
			qlog.fileLogger.WithFields(fields).Error(fmt.Sprintf("SERVFAIL:[%s]", result.Message))
		}
		if qlog.stdLogger != nil {
			builder.WriteString(fmt.Sprintf("SERVFAIL:[%s]", result.Message))
			qlog.stdLogger.Error(builder.String())
		}

		return
	} else {
		builder.WriteString(fmt.Sprintf("RESPONSE[%s]", dns.RcodeToString[response.Rcode]))
	}

	// output built string
	if qlog.fileLogger != nil {
		qlog.fileLogger.WithFields(fields).Info(dns.RcodeToString[response.Rcode])
	}
	if qlog.stdLogger != nil {
		qlog.stdLogger.Info(builder.String())
	}
}

func (qlog *qlog) getReverseName(address string) string {
	// look in local cache for name
	if value, found := qlog.cache.Get(address); found {
		if valueString, ok := value.(string); ok {
			return valueString
		}
	}

	name := ""

	// if there is a reverselookup function use it to add a reverse lookup step
	if qlog.rlookup != nil {
		name = qlog.rlookup(address)
		if strings.HasSuffix(name, ".") {
			name = name[:len(name)-1]
		}
	}

	// look in the mdns cache 
	if qlog.mdnsCache != nil {
		name := ReadCachedHostname(qlog.mdnsCache, address)
		if name != "" {
			return name
		}
	}

	// if no result from rlookup then try and lookup the netbios name from the host
	if "" == name {
		var err error
		name, err = util.LookupNetBIOSName(address)
		if err != nil {
			// don't really need to see these
			log.Tracef("During NETBIOS lookup: %s", err)
		}
	}

	// store result, even empty results, to prevent continual lookups
	qlog.cache.Set(address, name, cache.DefaultExpiration)

	return name
}

// this is the actual log worker that handles incoming log messages in a separate go routine
func (qlog *qlog) logWorker() {
	// create ticker from conf
	duration, _ := util.ParseDuration(qlog.qlConf.BatchInterval)
	ticker := time.NewTicker(duration)
	// prune every hour (also prunes on startup)
	pruneTicker := time.NewTicker(1 * time.Hour)

	// stop the timer immediately if we aren't persisting records
	if !*(qlog.qlConf.Persist) {
		ticker.Stop()
	}

	// loop until...
	for {
		select {
		case info := <-qlog.logInfoChan:
			if info != nil {
				// condition the log info item in this thread
				if info.Request != nil && len(info.Request.Question) > 0 {
					info.RequestDomain = info.Request.Question[0].Name
					info.RequestType = dns.Type(info.Request.Question[0].Qtype).String()
				}

				if info.Response != nil {
					answerValues := util.GetAnswerValues(info.Response)
					if len(answerValues) > 0 {
						info.ResponseText = answerValues[0]
					}
				}

				if info.Result != nil {
					info.Consumer = info.Result.Consumer
					if info.Result.Blocked {
						info.Blocked = true
						if info.Result.BlockedList != nil {
							info.BlockedList = info.Result.BlockedList.CanonicalName()
						}
						info.BlockedRule = info.Result.BlockedRule
					}
				}

				if info.RequestContext != nil {
					info.ConnectionType = info.RequestContext.Protocol
				}

				// get reverse lookup name
				info.ClientName = qlog.getReverseName(info.Address)
			}

			// only log to
			if info != nil && ("" != qlog.qlConf.File || *(qlog.qlConf.Stdout)) {
				qlog.log(info)
			}
			// only persist if configured, which is default
			if *(qlog.qlConf.Persist) {
				qlog.logDB(info, info == nil)
			}
		case <-qlog.doneChan:
			break
		case <-ticker.C:
			qlog.logDB(nil, true)
		case <-pruneTicker.C:
			qlog.prune()
		}
	}

	// stop tickers
	ticker.Stop()
	pruneTicker.Stop()
}

func (qlog *qlog) Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	// create message for sending to various endpoints
	msg := new(LogInfo)
	msg.Address = address.String()
	msg.Request = request
	msg.Response = response
	msg.Result = result
	msg.RequestContext = rCon
	msg.Created = time.Now()
	// put on channel
	qlog.logInfoChan <- msg
}

func (qlog *qlog) Query(query *QueryLogQuery) []LogInfo {
	// select entries from qlog
	selectStmt := "SELECT Address, ClientName, Consumer, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created FROM qlog"

	// so we can dynamically build the where clause
	whereClauses := make([]string, 0)
	whereValues := make([]interface{}, 0)

	// result holding
	var rows *sql.Rows
	var err error

	// build query
	if query.After != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("Created > $%d", len(whereClauses)+1))
		whereValues = append(whereValues, query.After)
	}
	if query.Before != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("Created < $%d", len(whereClauses)+1))
		whereValues = append(whereValues, query.Before)
	}

	if "" != query.Address {
		whereClauses = append(whereClauses, fmt.Sprintf("Address = $%d", len(whereClauses)+1))
		whereValues = append(whereValues, query.Address)
	}

	if "" != query.RequestDomain {
		whereClauses = append(whereClauses, fmt.Sprintf("RequestDomain = $%d", len(whereClauses)+1))
		whereValues = append(whereValues, query.RequestDomain)
	}

	if query.Blocked != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("Blocked = $%d", len(whereClauses)+1))
		whereValues = append(whereValues, query.Blocked)
	}

	// finalize query part
	if len(whereClauses) > 0 {
		selectStmt = selectStmt + " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// sort
	sortBy := "Created"
	sortReversed := query.Reverse
	direction := "ASC"
	if "" != query.SortBy && util.StringIn(strings.ToLower(query.SortBy), validSorts) {
		sortBy = query.SortBy
	}
	if "created" == strings.ToLower(sortBy) {
		direction = "DESC"
	}
	if sortReversed != nil && *sortReversed == true {
		if "DESC" == direction {
			direction = "ASC"
		} else if "ASC" == direction {
			direction = "DESC"
		}
	}
	// add sort
	selectStmt = selectStmt + fmt.Sprintf(" ORDER BY %s %s", sortBy, direction)

	// set limits
	if query.Limit > 0 {
		selectStmt = selectStmt + fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Skip > 0 {
		selectStmt = selectStmt + fmt.Sprintf(" OFFSET %d", query.Skip)
	}
	// make query
	rows, err = qlog.store.Query(selectStmt, whereValues...)
	defer rows.Close()

	// if rows is nil return empty array
	if rows == nil || err != nil {
		if err != nil {
			log.Errorf("Query log query failed: %s", err)
		}
		return []LogInfo{}
	}

	// otherwise create an array of the required size
	results := make([]LogInfo, 0)

	// only define once
	var clientName sql.NullString
	var consumer sql.NullString

	for rows.Next() {
		info := LogInfo{}
		err = rows.Scan(&info.Address, &clientName, &consumer, &info.RequestDomain, &info.RequestType, &info.ResponseText, &info.Blocked, &info.BlockedList, &info.BlockedRule, &info.Created)
		if err != nil {
			log.Errorf("Scanning qlog results: %s", err)
			continue
		}

		// add potentially nil values separately
		if clientName.Valid {
			info.ClientName = clientName.String
		}
		if consumer.Valid {
			info.Consumer = consumer.String
		}

		results = append(results, info)
	}

	return results
}

func (qlog *qlog) Stop() {
	// flush batches
	qlog.logInfoChan <- nil
	// be done
	qlog.doneChan <- true
	// prune old records
	qlog.prune()
	// close db
	qlog.store.Close()
}
