package qlog
//go:generate codecgen -r LogInfo -o qlog_gen.go qlog.go

import (
	"fmt"
	"net"
	"os"
	"path"
	"strings"
	"time"

	"github.com/dgraph-io/badger"
	"github.com/miekg/dns"
	"github.com/timshannon/badgerhold"
	"github.com/ugorji/go/codec"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/resolver"
)

// info passed over channel and stored in database
// and that is recovered via the Query method
type LogInfo struct {
	// original values
	Address  		string  

	// hold the information but aren't usually serialized
	Request  		*dns.Msg 					`codec:"-",json:"-"`
	Response 		*dns.Msg 					`codec:"-",json:"-"`
	Result   		*resolver.ResolutionResult  `codec:"-",json:"-"`
	RequestContext  *resolver.RequestContext    `codec:"-",json:"-"`

	// generated/calculated values
	ConnectionType  string  
	RequestDomain   string  
	RequestType     string
	Blocked         bool
	BlockedList     string
	BlockedRule     string
	Created  		time.Time
}

// the type that is used to make queries against the
// query log (should be used by the web interface to
// find queries)
type QueryLogQuery struct {
	// query on fields
	Address         string
	ConnectionType  string 
	RequestDomain   string
	RequestType     string
	Blocked         *bool
	// query on created time
	After           *time.Time
	Before          *time.Time
	// query limits for paging
	Skip			int
	Limit           int
	// query sort
	SortBy          string
	Reverse         *bool
}

// store database location
type qlog struct {
	store		*badgerhold.Store
	qlConf 		*config.GudgeonQueryLog
	logInfoChan chan *LogInfo
}

// public interface
type QLog interface {
	Query(query *QueryLogQuery) []LogInfo
	Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult)
}


type coder struct {
	handle  codec.Handle
}

func (coder *coder) customEncode(value interface{}) ([]byte, error) {
	var data []byte
	enc := codec.NewEncoderBytes(&data, coder.handle)
	err := enc.Encode(value)
	return data, err
}

func (coder *coder) customDecode(data []byte, value interface{}) error {
	dec := codec.NewDecoderBytes(data, coder.handle)
	return dec.Decode(value)
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
	qlog.logInfoChan = make(chan *LogInfo)
	go qlog.logWorker()

	// get path to long-standing data ({home}/'data') and make sure it exists
	dataDir := conf.DataRoot()
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}

	// open db
	dbDir := path.Join(dataDir, "query_log")
	options := badgerhold.DefaultOptions

	// set encode/decode
	coder := &coder{}
	coder.handle = &codec.MsgpackHandle{}
	options.Encoder = coder.customEncode
	options.Decoder = coder.customDecode

	// reduce memory consumption
	options.MaxTableSize = 64 << 12
	options.NumMemtables = 1
	
	// set where to output data
	options.Dir = dbDir
	options.ValueDir = dbDir

	// don't log through badger logging
	options.Logger = nil

	store, err := badgerhold.Open(options)
	if err != nil {
		return nil, err
	}

	// keep pointer to store
	qlog.store = store

	return qlog, nil
}

func (qlog *qlog) logDB(info *LogInfo) {
	// clean up stuff
	if info.Request != nil {

	}

	err := qlog.store.Badger().Update(func(tx * badger.Txn) error {
		err := qlog.store.TxInsert(tx, badgerhold.NextSequence(), info)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error saving log info to db: %s\n", err)
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

	// log result if found
	logPrefix := fmt.Sprintf("[%s/%s] q:|%s|%s|->", address, rCon.Protocol, domain, requestType)
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
		qlog.logDB(info)
	}
}

func (qlog *qlog) Log(address *net.IP, request *dns.Msg, response *dns.Msg, rCon *resolver.RequestContext, result *resolver.ResolutionResult) {
	// create message for sending to various endpoints
	msg := new(LogInfo)
	msg.Address = address.String()
	msg.Request = request
	if request != nil && len(request.Question) > 0 {
		msg.RequestDomain = request.Question[0].Name
		msg.RequestType = dns.Type(request.Question[0].Qtype).String()
	}
	msg.Response = response
	msg.Result = result
	if result != nil && result.Blocked {
		msg.Blocked = true
		if result.BlockedList != nil {
			msg.BlockedList = result.BlockedList.CanonicalName()
		}
		msg.BlockedRule = result.BlockedRule
	}
	msg.RequestContext = rCon
	msg.ConnectionType = rCon.Protocol
	msg.Created = time.Now()
	// put on channel
	qlog.logInfoChan <- msg
}

func (qlog *qlog) Query(query *QueryLogQuery) []LogInfo {
	// result holder
	var result []LogInfo

	// create query
	bhq := &badgerhold.Query{}

	if "" != query.Address {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("Address").Eq(query.Address)
		} else {
			bhq = bhq.And("Address").Eq(query.Address)
		}
	}

	if "" != query.ConnectionType {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("ConnectionType").Eq(query.ConnectionType)
		} else {
			bhq = bhq.And("ConnectionType").Eq(query.ConnectionType)
		}
	}

	if "" != query.RequestDomain {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("RequestDomain").Eq(query.RequestDomain)
		} else {
			bhq = bhq.And("RequestDomain").Eq(query.RequestDomain)
		}
	}

	if "" != query.RequestType {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("RequestType").Eq(query.RequestType)
		} else {
			bhq = bhq.And("RequestType").Eq(query.RequestType)
		}
	}

	if nil != query.Blocked {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("Blocked").Eq(query.Blocked)
		} else {
			bhq = bhq.And("Blocked").Eq(query.Blocked)
		}
	}

	if nil != query.After {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("Created").Gt(query.After)
		} else {
			bhq = bhq.And("Created").Gt(query.After)		
		}		
	}

	if nil != query.Before {
		if bhq.IsEmpty() {
			bhq = badgerhold.Where("Created").Lt(query.Before)
		} else {
			bhq = bhq.And("Created").Lt(query.Before)
		}
	}

	// set limits and skip counts (paging)
	if query.Skip > 0 {
		bhq.Skip(query.Skip)
	}

	if query.Limit > 0 {
		bhq.Limit(query.Limit)
	}

	// set sort order
	if "" == query.SortBy {
		bhq = bhq.SortBy("Created")
	} else {
		bhq = bhq.SortBy(query.SortBy)
	}

	// reverse by default, to match "created" default search
	// sure this will be a problem later though because most
	// searches want the ascending (non-reverse) order
	if query.Reverse == nil || (*query.Reverse) == true {
		bhq = bhq.Reverse()
	}

	// do query
	err := qlog.store.Find(&result, bhq)
	if err != nil {
		fmt.Printf("Error finding: %s\n", err)
	}

	return result
}