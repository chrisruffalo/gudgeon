package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// lit of valid sort names (lower case for ease of use with util.StringIn)
var validSorts = []string{"address", "connectiontype", "requestdomain", "requesttype", "blocked", "blockedlist", "blockedrule", "created"}

const bufferFlushStmt = "INSERT INTO qlog (Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Rcode, Cached, Blocked, Match, MatchList, MatchRule, ServiceTime, Created, EndTime) SELECT Address, Consumer, ClientName, RequestDomain, RequestType, ResponseText, Rcode, Cached, Blocked, Match, MatchList, MatchRule, ServiceTime, Created, EndTime FROM buffer WHERE true"

// allows a dependency injection-way of defining a reverse lookup function, takes a string address (should be an IP) and returns a string that contains the domain name result
type ReverseLookupFunction = func(address string) string

// the type that is used to make queries against the
// query log (should be used by the web interface to
// find queries)
type QueryLogQuery struct {
	// query on fields
	Address        string
	ClientName     string
	ConnectionType string
	RequestDomain  string
	RequestType    string
	ResponseText   string
	Blocked        *bool
	Cached         *bool
	// aspects of the match
	Match     *rule.Match
	MatchList string
	MatchRule string
	// query on created time
	After  *time.Time
	Before *time.Time
	// query limits for paging
	Skip  int
	Limit int
	// query sort
	SortBy    string
	Direction string
}

// store database location
type qlog struct {
	qlConf *config.GudgeonQueryLog
	db     *sql.DB

	duration time.Duration

	fileLogger *log.Logger
	stdLogger  *log.Logger
}

// public interface
type QueryLog interface {
	Query(query *QueryLogQuery) ([]*InfoRecord, uint64)
	QueryFunc(query *QueryLogQuery, accumulator QueryAccumulator)
	Stop()

	// package management methods
	log(info *InfoRecord)
	flush(tx *sql.Tx)
	prune(tx *sql.Tx)
}

// create a new query log according to configuration
func NewQueryLog(conf *config.GudgeonConfig, db *sql.DB) (QueryLog, error) {
	qlConf := conf.QueryLog
	if qlConf == nil || !(*qlConf.Enabled) {
		return nil, nil
	}

	// create new empty qlog
	qlog := &qlog{
		qlConf: qlConf,
	}

	if *(qlConf.Persist) {
		qlog.db = db
	}

	// parse duration
	qlog.duration, _ = util.ParseDuration(qlog.qlConf.Duration)

	// create distinct loggers for query output
	if qlConf.File != "" {
		// create destination and writer
		dirpart := path.Dir(qlConf.File)
		if _, err := os.Stat(dirpart); os.IsNotExist(err) {
			err = os.MkdirAll(dirpart, os.ModePerm)
			if err != nil {
				log.Errorf("While creating path for query log output file: %s", err)
			}
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

	return qlog, nil
}

func (qlog *qlog) prune(tx *sql.Tx) {
	_, err := tx.Exec("DELETE FROM qlog WHERE Created <= ?", time.Now().Add(-1*qlog.duration))
	if err != nil {
		log.Errorf("Error pruning query log data: %s", err)
	}
}

func (qlog *qlog) flush(tx *sql.Tx) {
	_, err := tx.Exec(bufferFlushStmt)
	if err != nil {
		log.Errorf("Could not flush query log data: %s", err)
		return
	}
}

func (qlog *qlog) log(info *InfoRecord) {
	// don't log if stdout is off and the file isn't specified
	if !(*qlog.qlConf.Stdout) && qlog.qlConf.File == "" {
		return
	}

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
		if fields != nil {
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
	if fields != nil {
		fields["address"] = info.Address
		fields["protocol"] = rCon.Protocol
		fields["consumer"] = info.Consumer
		fields["requestDomain"] = info.RequestDomain
		fields["requestType"] = info.RequestType
		fields["cached"] = false
		fields["rcode"] = info.Rcode
	}

	if response.Rcode == dns.RcodeServerFailure {
		// write as error and return
		if qlog.fileLogger != nil {
			qlog.fileLogger.WithFields(fields).Error(fmt.Sprintf("SERVFAIL:[%s]", result.Message))
		}
		if qlog.stdLogger != nil {
			builder.WriteString(fmt.Sprintf("SERVFAIL:[%s]", result.Message))
			qlog.stdLogger.Error(builder.String())
		}

		return
	} else if result != nil && response.Rcode != dns.RcodeNameError {
		if result.Blocked {
			builder.WriteString("BLOCKED")
		} else if result.Match == rule.MatchBlock {
			builder.WriteString("RULE BLOCKED")
			if fields != nil {
				fields["match"] = result.Match
				fields["matchType"] = "BLOCKED"
			}
			if result.MatchList != nil {
				builder.WriteString("[")
				builder.WriteString(result.MatchList.CanonicalName())
				if fields != nil {
					fields["matchList"] = result.MatchList.CanonicalName()
				}
				if result.MatchRule != "" {
					builder.WriteString("|")
					builder.WriteString(result.MatchRule)
					if fields != nil {
						fields["matchRule"] = result.MatchRule
					}
				}
				builder.WriteString("]")
			}
		} else {
			if result.Match == rule.MatchAllow {
				if fields != nil {
					fields["match"] = result.Match
					fields["matchType"] = "ALLOWED"
				}
			}
			if result.Cached {
				builder.WriteString("c:[")
				builder.WriteString(result.Resolver)
				builder.WriteString("]")
				if fields != nil {
					fields["resolver"] = result.Resolver
					fields["cached"] = "true"
				}
			} else {
				builder.WriteString("r:[")
				builder.WriteString(result.Resolver)
				builder.WriteString("]")
				builder.WriteString("->")
				builder.WriteString("s:[")
				builder.WriteString(result.Source)
				builder.WriteString("]")
				if fields != nil {
					fields["resolver"] = result.Resolver
					fields["source"] = result.Source
				}
			}

			builder.WriteString("->")

			if len(response.Answer) > 0 {
				answerValues := util.GetAnswerValues(response)
				if len(answerValues) > 0 {
					builder.WriteString(answerValues[0])
					if fields != nil {
						fields["answer"] = answerValues[0]
					}
					if len(answerValues) > 1 {
						builder.WriteString(fmt.Sprintf(" (+%d)", len(answerValues)-1))
					}
				} else {
					builder.WriteString("(EMPTY RESPONSE)")
					if fields != nil {
						fields["answer"] = "<< EMPTY >>"
					}
				}
			} else {
				builder.WriteString("(NO INFO RESPONSE)")
				if fields != nil {
					fields["answer"] = "<< NONE >>"
				}
			}
		}
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

type QueryAccumulator = func(count uint64, info *InfoRecord)

func (qlog *qlog) QueryFunc(query *QueryLogQuery, accumulator QueryAccumulator) {
	if nil == qlog.db {
		return
	}

	// select entries from qlog
	selectStmt := "SELECT Address, ClientName, Consumer, RequestDomain, RequestType, ResponseText, Rcode, Blocked, Match, MatchList, MatchRule, Cached, ServiceTime, Created, EndTime FROM qlog"
	countStmt := "SELECT COUNT(*) FROM qlog"

	// so we can dynamically build the where clause
	orClauses := []string{"1 = 1"}
	whereClauses := []string{"1 = 1"}
	orValues := make([]interface{}, 0)
	whereValues := make([]interface{}, 0)

	// result holding
	var rows *sql.Rows
	var err error

	// or clause
	if "" != query.Address {
		orClauses = append(orClauses, "Address like ?")
		orValues = append(orValues, "%"+query.Address+"%")
	}

	if "" != query.ClientName {
		orClauses = append(orClauses, "ClientName like ?")
		orValues = append(orValues, "%"+query.ClientName+"%")
	}

	if "" != query.RequestDomain {
		orClauses = append(orClauses, "RequestDomain like ?")
		orValues = append(orValues, "%"+query.RequestDomain+"%")
	}

	if "" != query.ResponseText {
		orClauses = append(orClauses, "ResponseText like ?")
		orValues = append(orValues, "%"+query.ResponseText+"%")
	}

	if query.Blocked != nil {
		whereClauses = append(whereClauses, "Blocked = ?")
		whereValues = append(whereValues, query.Blocked)
	}

	if query.Match != nil {
		whereClauses = append(whereClauses, "Match = ?")
		whereValues = append(whereValues, query.Match)
	}

	if "" != query.MatchList {
		whereClauses = append(whereClauses, "MatchList like ?")
		whereValues = append(whereValues, query.MatchList)
	}

	if "" != query.MatchRule {
		whereClauses = append(whereClauses, "MatchRule like ?")
		whereValues = append(whereValues, query.MatchRule)
	}

	if query.Cached != nil {
		whereClauses = append(whereClauses, "Cached = ?")
		whereValues = append(whereValues, query.Cached)
	}

	if query.After != nil {
		whereClauses = append(whereClauses, "Created > ?")
		whereValues = append(whereValues, query.After)
	}

	if query.Before != nil {
		whereClauses = append(whereClauses, "Created < ?")
		whereValues = append(whereValues, query.Before)
	}

	// finalize query part
	if len(whereClauses) > 0 || len(orClauses) > 0 {
		if len(orClauses) > 1 {
			orClauses = orClauses[1:]
		}
		if len(whereClauses) > 1 {
			whereClauses = whereClauses[1:]
		}

		clauses := strings.Join([]string{"(" + strings.Join(orClauses, " OR ") + ")", strings.Join(whereClauses, " AND ")}, " AND ")
		selectStmt = selectStmt + " WHERE " + clauses
		// copy current select statement to use for length query if needed
		countStmt = countStmt + " WHERE " + clauses
	}

	// add or/and values together
	whereValues = append(orValues, whereValues...)

	// sort and sort direction
	sortBy := "created"
	direction := strings.ToUpper(query.Direction)
	if !util.StringIn(direction, []string{"DESC", "ASC"}) {
		direction = ""
	}
	if "" != query.SortBy && util.StringIn(strings.ToLower(query.SortBy), validSorts) {
		sortBy = query.SortBy
	}
	if "created" == strings.ToLower(sortBy) && "" == direction {
		direction = "DESC"
	} else if "" == direction {
		direction = "ASC"
	}

	// add sort
	selectStmt = selectStmt + fmt.Sprintf(" ORDER BY %s %s", sortBy, direction)

	// default length of query is 0
	resultLen := uint64(0)

	// set limits
	if query.Limit > 0 {
		selectStmt = selectStmt + fmt.Sprintf(" LIMIT %d", query.Limit)
	}
	if query.Skip > 0 {
		selectStmt = selectStmt + fmt.Sprintf(" OFFSET %d", query.Skip)
	}

	err = qlog.db.QueryRow(countStmt, whereValues...).Scan(&resultLen)
	if err != nil {
		log.Errorf("Could not get log item count: %s", err)
	}
	if err != nil {
		log.Errorf("Could not close prepared statement: %s", err)
		return
	}

	rows, err = qlog.db.Query(selectStmt, whereValues...)
	if err != nil {
		log.Errorf("Query log query failed: %s", err)
		return
	}

	// if rows is nil return empty array
	if rows == nil {
		return
	}

	// scan each row and get results
	info := &InfoRecord{}
	for rows.Next() {
		err = rows.Scan(&info.Address, &info.ClientName, &info.Consumer, &info.RequestDomain, &info.RequestType, &info.ResponseText, &info.Rcode, &info.Blocked, &info.Match, &info.MatchList, &info.MatchRule, &info.Cached, &info.ServiceMilliseconds, &info.Created, &info.Finished)
		if err != nil {
			log.Errorf("Scanning qlog results: %s", err)
			continue
		}
		accumulator(resultLen, info)
	}

	err = rows.Close()
	if err != nil {
		log.Errorf("Could not close row set: %s", err)
	}
}

func (qlog *qlog) Query(query *QueryLogQuery) ([]*InfoRecord, uint64) {
	var records []*InfoRecord
	var totalCount uint64

	qlog.QueryFunc(query, func(count uint64, info *InfoRecord) {
		if count > 0 {
			totalCount = count
		}
		if info != nil {
			records = append(records, &InfoRecord{
				ServiceMilliseconds: info.ServiceMilliseconds,
				Request:             info.Request,
				Created:             info.Created,
				Finished:            info.Finished,
				MatchRule:           info.MatchRule,
				MatchList:           info.MatchList,
				Blocked:             info.Blocked,
				RequestContext:      info.RequestContext,
				Address:             info.Address,
				Cached:              info.Cached,
				ClientName:          info.ClientName,
				ConnectionType:      info.ConnectionType,
				Consumer:            info.Consumer,
				Match:               info.Match,
				MatchListShort:      info.MatchListShort,
				Rcode:               info.Rcode,
				RequestDomain:       info.RequestDomain,
				RequestType:         info.RequestType,
				Response:            info.Response,
				ResponseText:        info.ResponseText,
				Result:              info.Result,
			})
		}
	})

	return records, totalCount
}

func (qlog *qlog) Stop() {

}
