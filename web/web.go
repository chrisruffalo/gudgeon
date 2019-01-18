package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	metrics "github.com/rcrowley/go-metrics"

	"github.com/chrisruffalo/gudgeon/config"
)

type web struct {
}

type Web interface {
	Serve(conf *config.GudgeonConfig) error
}

func New() Web {
	return &web{}
}

// get metrics counter named in query
func (web *web) GetCounter(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string, 0)
	response["count"] = "0"

	params := mux.Vars(r)
	if params["counter-name"] != "" {
		item := metrics.DefaultRegistry.Get(params["counter-name"])
		if counter, ok := item.(metrics.Counter); ok && counter != nil {
			response["count"] = fmt.Sprintf("%d", counter.Count())
		}
	}
	
	json.NewEncoder(w).Encode(response)
}

func (web *web) GetMeter(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string, 0)
	response["count"] = "0"
	response["rate1"] = "0.0000"
	response["rate5"] = "0.0000"
	response["rate15"] = "0.0000"

	params := mux.Vars(r)
	if params["meter-name"] != "" {
		item := metrics.DefaultRegistry.Get(params["meter-name"])
		if meter, ok := item.(metrics.Meter); ok && meter != nil {
			response["count"] = fmt.Sprintf("%d", meter.Count())	
			response["rate1"] = fmt.Sprintf("%f", meter.Rate1())	
			response["rate5"] = fmt.Sprintf("%f", meter.Rate5())
			response["rate15"] = fmt.Sprintf("%f", meter.Rate15())
		}
	}
	
	json.NewEncoder(w).Encode(response)
}

func (web *web) GetGauge(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string, 0)
	response["value"] = "0"

	params := mux.Vars(r)
	if params["gauge-name"] != "" {
		item := metrics.DefaultRegistry.Get(params["gauge-name"])
		if gauge, ok := item.(metrics.Gauge); ok && gauge != nil {
			response["value"] = fmt.Sprintf("%d", gauge.Value())	
		}
	}
	
	json.NewEncoder(w).Encode(response)
}

func (web *web) Serve(conf *config.GudgeonConfig) error {
	// create new router
	router := mux.NewRouter()

	// attach metrics
	router.HandleFunc("/web/api/metrics/counter/{counter-name}", web.GetCounter).Methods("GET")
	router.HandleFunc("/web/api/metrics/meter/{meter-name}", web.GetMeter).Methods("GET")
	router.HandleFunc("/web/api/metrics/gauge/{gauge-name}", web.GetGauge).Methods("GET")

	// attach to static assets
	router.PathPrefix("/").Handler(http.FileServer(rice.MustFindBox("assets").HTTPBox()))
	
	// go serve
	webConf := conf.Web
	go http.ListenAndServe(fmt.Sprintf("%s:%d", webConf.Address, webConf.Port), router)
	fmt.Printf("Started web ui on %s:%d\n", webConf.Address, webConf.Port)

	return nil
}
