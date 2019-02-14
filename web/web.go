package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"github.com/json-iterator/go"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/qlog"
)

type web struct {
	metrics  metrics.Metrics
	queryLog qlog.QLog
}

type Web interface {
	Serve(conf *config.GudgeonConfig, metrics metrics.Metrics, qlog qlog.QLog) error
}

func New() Web {
	return &web{}
}

// write all metrics out to encoder
var json = jsoniter.Config{
	EscapeHTML:                    false,
	MarshalFloatWith6Digits:       true, // will lose precession
	ObjectFieldMustBeSimpleString: true, // do not unescape object field
	SortMapKeys:                   true,
	ValidateJsonRawMessage:        true,
	DisallowUnknownFields:         false,
}.Froze()

// get metrics counter named in query
func (web *web) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if web.metrics == nil {
		http.Error(w, "Metrics not enabled", http.StatusNotFound)
		return
	}

	// get all available metrics
	response := web.metrics.GetAll()

	json.NewEncoder(w).Encode(response)
}

func (web *web) GetQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	if web.queryLog == nil {
		http.Error(w, "Query Log not enabled", http.StatusNotFound)
		return
	}

	// set default query options
	query := &qlog.QueryLogQuery{}
	if query.Limit < 1 {
		query.Limit = 100 // default limit to 100 entries
	}

	// add in other query options from params
	vals := r.URL.Query()

	if limit, ok := vals["limit"]; ok && len(limit) > 0 {
		if "none" == strings.ToLower(limit[0]) {
			query.Limit = 0
		} else {
			iLimit, err := strconv.Atoi(limit[0])
			if err == nil {
				query.Limit = iLimit
			}
		}
	}

	if blocked, ok := vals["blocked"]; ok && len(blocked) > 0 {
		bl := blocked[0]
		if "true" == strings.ToLower(bl) {
			boolHolder := true
			query.Blocked = &boolHolder
		} else if "false" == strings.ToLower(bl) {
			boolHolder := false
			query.Blocked = &boolHolder
		}
	}

	if requestDomains, ok := vals["rdomain"]; ok && len(requestDomains) > 0 {
		query.RequestDomain = requestDomains[0]
	}

	if after, ok := vals["after"]; ok && len(after) > 0 {

	}

	// query against query log
	results := web.queryLog.Query(query)

	if len(results) == 0 {
		http.Error(w, "[]", http.StatusNotFound)
		return
	}

	// return encoded results
	json.NewEncoder(w).Encode(results)
}

func (web *web) Serve(conf *config.GudgeonConfig, metrics metrics.Metrics, qlog qlog.QLog) error {
	// set metrics endpoint
	web.metrics = metrics
	web.queryLog = qlog

	// create new router
	router := mux.NewRouter()

	// attach metrics
	router.HandleFunc("/api/metrics", web.GetMetrics).Methods("GET")
	router.HandleFunc("/api/log", web.GetQueryLogInfo).Methods("GET")

	// attach to static assets
	router.PathPrefix("/").Handler(http.FileServer(rice.MustFindBox("assets").HTTPBox()))

	// go serve
	webConf := conf.Web
	go http.ListenAndServe(fmt.Sprintf("%s:%d", webConf.Address, webConf.Port), router)
	fmt.Printf("Started web ui on %s:%d\n", webConf.Address, webConf.Port)

	return nil
}
