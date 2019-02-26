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

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	// max batch size allowed
	qlogInsertBatchSize = 50

	// constant insert statement
	qlogInsertStatement = "insert into qlog (Address, Consumer, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
)
var qlogInsertBatchTime = 1 * time.Second

// lit of valid sort names (lower case for ease of use with util.StringIn)
var validSorts = []string{"address", "connectiontype", "requestdomain", "requesttype", "blocked", "blockedlist", "blockedrule", "created"}

// info passed over channel and stored in database
// and that is recovered via the Query method
type LogInfo struct {
	// original values
	Address string

	// hold the information but aren't serialized
	Request        *dns.Msg                   `json:"-"`
	Response       *dns.Msg                   `json:"-"`
	Result         *resolver.ResolutionResult `json:"-"`
	RequestContext *resolver.RequestContext   `json:"-"`

	// generated/calculated values
	Consumer       string
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
	store       *sql.DB
	qlConf      *config.GudgeonQueryLog
	logInfoChan chan *LogInfo
	batch       []*LogInfo
}

// public interface
type QLog interface {
	Query(query *QueryLogQuery) []LogInfo
	Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)
}

// create a new query log according to configuration
func New(conf *config.GudgeonConfig) (QLog, error) {
	qlConf := conf.QueryLog
	if qlConf == nil || !*(qlConf.Enabled) {
		return nil, nil
	}

	// create new empty qlog
	qlog := &qlog{}
	qlog.qlConf = qlConf

	// create log channel
	qlog.batch = make([]*LogInfo, 0, qlogInsertBatchSize)
	qlog.logInfoChan = make(chan *LogInfo, qlogInsertBatchSize*10)
	go qlog.logWorker()

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
		migrationsBox := rice.MustFindBox("migrations")

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
	}

	return qlog, nil
}

func (qlog *qlog) logDB(info *LogInfo, forceInsert bool) {
	// only add to batch if not nil
	if info != nil {
		qlog.batch = append(qlog.batch, info)
	}

	// insert whole batch and reset batch
	if (forceInsert || len(qlog.batch) >= qlogInsertBatchSize) && len(qlog.batch) > 0 {
		// attempt to insert statements
		stmt := qlogInsertStatement + strings.Repeat(", (?, ?, ?, ?, ?, ?, ?, ?, ?)", len(qlog.batch)-1)
		vars := make([]interface{}, 0, len(qlog.batch)*9)
		for _, i := range qlog.batch {
			vars = append(vars, i.Address)
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
			fmt.Printf("Error preparing statement: %s\n", err)
		} else {
			_, err = pstmt.Exec(vars...)
			if err != nil {
				fmt.Printf("Could not insert into db: %s\n", err)
			}
		}

		// remake batch for inserting
		qlog.batch = make([]*LogInfo, 0, qlogInsertBatchSize)
	}
}

func (qlog *qlog) logStdout(info *LogInfo) {
	// get values
	address := info.Address
	domain := info.RequestDomain
	requestType := info.RequestType
	response := info.Response
	result := info.Result
	rCon := info.RequestContext
	consumerName := info.Consumer

	// log result if found
	logPrefix := fmt.Sprintf("[%s/%s|%s] q:|%s|%s|->", address, rCon.Protocol, consumerName, domain, requestType)
	if result != nil {
		logSuffix := "->"
		if result.Blocked {
			if result.BlockedList != nil {
				listName := result.BlockedList.CanonicalName()
				ruleText := result.BlockedRule
				if ruleText != "" {
					fmt.Printf("%s BLOCKED[%s|%s]\n", logPrefix, listName, ruleText)
				} else {
					fmt.Printf("%s BLOCKED[%s]\n", logPrefix, listName)
				}
			} else {
				fmt.Printf("%s BLOCKED\n", logPrefix)
			}
		} else {
			if len(response.Answer) > 0 {
				answerValues := util.GetAnswerValues(response)
				if len(answerValues) > 0 {
					logSuffix += answerValues[0]
					if len(answerValues) > 1 {
						logSuffix += fmt.Sprintf(" (+%d)", len(answerValues)-1)
					}
				} else {
					logSuffix += "(EMPTY RESPONSE)"
				}
			}

			// nothing appended so look at SOA
			if strings.TrimSpace(logSuffix) == "->" {
				if len(response.Ns) > 0 && response.Ns[0].Header().Rrtype == dns.TypeSOA && len(response.Ns[0].String()) > 0 {
					logSuffix += response.Ns[0].(*dns.SOA).Ns
					if len(response.Ns) > 1 {
						logSuffix += fmt.Sprintf(" (+%d)", len(response.Ns)-1)
					}
				} else {
					logSuffix += "(EMPTY)"
				}
			}

			if result.Cached {
				fmt.Printf("%sc:[%s]%s\n", logPrefix, result.Resolver, logSuffix)
			} else {
				fmt.Printf("%sr:[%s]->s:[%s]%s\n", logPrefix, result.Resolver, result.Source, logSuffix)
			}
		}
	} else if response.Rcode == dns.RcodeServerFailure {
		fmt.Printf("%s SERVFAIL:[%s]\n", logPrefix, result.Message)
	} else {
		fmt.Printf("%s RESPONSE[%s]\n", logPrefix, dns.RcodeToString[response.Rcode])
	}
}

// this is the actual log worker that handles incoming log messages in a separate go routine
func (qlog *qlog) logWorker() {
	// create ticker
	ticker := time.NewTicker(qlogInsertBatchTime)

	// loop until...
	for {
		select {
		case info := <- qlog.logInfoChan:
			// only log to stdout if configured
			if info != nil && *(qlog.qlConf.Stdout) {
				qlog.logStdout(info)
			}
			// only persist if configured, which is default
			if *(qlog.qlConf.Persist) {
				qlog.logDB(info, info == nil)
			}
		case <- ticker.C:
			qlog.logDB(nil, true)
		}
	}
}

func (qlog *qlog) Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	// create message for sending to various endpoints
	msg := new(LogInfo)
	msg.Address = address.String()
	if request != nil && len(request.Question) > 0 {
		msg.Request = request.Copy()
		msg.RequestDomain = request.Question[0].Name
		msg.RequestType = dns.Type(request.Question[0].Qtype).String()
	}
	if response != nil {
		msg.Response = response.Copy()
		answerValues := util.GetAnswerValues(response)
		if len(answerValues) > 0 {
			msg.ResponseText = answerValues[0]
		}
	}
	msg.Result = result
	if result != nil {
		msg.Consumer = result.Consumer
		if result.Blocked {
			msg.Blocked = true
			if result.BlockedList != nil {
				msg.BlockedList = result.BlockedList.CanonicalName()
			}
			msg.BlockedRule = result.BlockedRule
		}
	}
	if rCon != nil {
		msg.RequestContext = rCon
		msg.ConnectionType = rCon.Protocol
	}
	msg.Created = time.Now()
	// put on channel
	qlog.logInfoChan <- msg
}

func (qlog *qlog) Query(query *QueryLogQuery) []LogInfo {
	// select entries from qlog
	selectStmt := "SELECT Address, RequestDomain, RequestType, ResponseText, Blocked, BlockedList, BlockedRule, Created FROM qlog"

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
			fmt.Printf("query: '%s'\n", selectStmt)
			if len(whereValues) > 0 {
				fmt.Printf("values: '%v'\n", whereValues)
			}
			fmt.Printf("error: %s\n", err)
		}
		return []LogInfo{}
	}

	// otherwise create an array of the required size
	results := make([]LogInfo, 0)

	// only define once
	var address string
	var requestDomain string
	var requestType string
	var responseText string
	var blocked bool
	var blockedList string
	var blockedRule string
	var created time.Time

	for rows.Next() {
		err = rows.Scan(&address, &requestDomain, &requestType, &responseText, &blocked, &blockedList, &blockedRule, &created)
		if err != nil {
			fmt.Printf("error scanning: %s\n", err)
			continue
		}
		logInfo := LogInfo{
			Address:       address,
			RequestDomain: requestDomain,
			RequestType:   requestType,
			ResponseText:  responseText,
			Blocked:       blocked,
			BlockedList:   blockedList,
			BlockedRule:   blockedRule,
			Created:       created,
		}
		results = append(results, logInfo)
	}

	return results
}
