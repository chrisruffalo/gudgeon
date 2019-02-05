package qlog

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path"
	"strings"

	"github.com/GeertJohan/go.rice"
	_ "github.com/mattn/go-sqlite3"
	"github.com/miekg/dns"
	"github.com/rubenv/sql-migrate"

	"github.com/chrisruffalo/gudgeon/config"
	gdb "github.com/chrisruffalo/gudgeon/db"
	"github.com/chrisruffalo/gudgeon/resolver"
)

// info passed over channel
type logInfo struct {
	address  *net.IP
	request  *dns.Msg
	response *dns.Msg
	result   *resolver.ResolutionResult
	rCon     *resolver.RequestContext
}

// store database location
type qlog struct {
	db 			*sql.DB
	qlConf 		*config.GudgeonQueryLog
	logInfoChan chan *logInfo
}

// public interface
type QLog interface {
	Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)
}

func New(conf *config.GudgeonConfig) (QLog, error) {

	qlConf := conf.QueryLog
	if qlConf == nil || !*(qlConf.Enabled) {
		return nil, nil
	}

	// create new empty qlog
	qlog := &qlog{}
	qlog.qlConf = qlConf
	qlog.logInfoChan = make(chan *logInfo)
	go qlog.logWorker()

	// get path to long-standing data ({home}/'data') and make sure it exists
	dataDir := conf.DataRoot()
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}

	// open database
	dbPath := path.Join(dataDir, "gudgeon-query-log.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("Could not open query log databse: %s\n", err)
	}
	qlog.db = db

	// get migrations
	box := rice.MustFindBox("qlog-migrations")
	migrationSource := gdb.NewMigrationSource(box)

	// do migration
	_, err = migrate.Exec(db, "sqlite3", migrationSource, migrate.Up)
	if err != nil {
		return nil, fmt.Errorf("Could not complete migration: %s\n", err)
	}

	return qlog, nil
}

func (qlog *qlog) logDb(info *logInfo) {

}


func (qlog *qlog) logStdout(info *logInfo) {
	// get values
	address := info.address
	request := info.request
	response := info.response
	result := info.result
	rCon := info.rCon

	// log result if found
	logPrefix := fmt.Sprintf("[%s/%s] q:|%s|%s|->", address.String(), rCon.Protocol, request.Question[0].Name, dns.Type(request.Question[0].Qtype).String())
	if result != nil {
		logSuffix := "->"
		if result.Blocked {
			listName := "UNKNOWN"
			if result.BlockedList != nil {
				listName = result.BlockedList.CanonicalName()
			}
			ruleText := result.BlockedRule
			fmt.Printf("%s BLOCKED[%s|%s]\n", logPrefix, listName, ruleText)
		} else {
			if len(response.Answer) > 0 {
				responseString := strings.TrimSpace(response.Answer[0].String())
				responseLen := len(responseString)
				headerString := strings.TrimSpace(response.Answer[0].Header().String())
				headerLen := len(headerString)
				if responseLen > 0 && headerLen < responseLen {
					logSuffix += strings.TrimSpace(responseString[headerLen:])
					if len(response.Answer) > 1 {
						logSuffix += fmt.Sprintf(" (+%d)", len(response.Answer)-1)
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
	for info := range qlog.logInfoChan {
		if *(qlog.qlConf.Stdout) {
			qlog.logStdout(info)
		}
		qlog.logDb(info)
	}
}


func (qlog *qlog) Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	// create message for sending to various endpoints
	msg := new(logInfo)
	msg.address = address
	msg.request = request
	msg.response = response
	msg.result = result
	msg.rCon = rCon
	// put on channel
	qlog.logInfoChan <- msg
}
