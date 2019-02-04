package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/metrics"
)

type web struct {
	metrics metrics.Metrics
}

type Web interface {
	Serve(conf *config.GudgeonConfig, metrics metrics.Metrics) error
}

func New() Web {
	return &web{}
}

// get metrics counter named in query
func (web *web) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if web.metrics == nil {
		http.Error(w, "Metrics not enabled", http.StatusNotFound)
		return
	}

	// get all available metrics
	response := web.metrics.GetAll()

	// write all metrics out to encoder
	json.NewEncoder(w).Encode(response)
}

func (web *web) Serve(conf *config.GudgeonConfig, metrics metrics.Metrics) error {
	// set metrics endpoint
	web.metrics = metrics

	// create new router
	router := mux.NewRouter()

	// attach metrics
	router.HandleFunc("/api/metrics", web.GetMetrics).Methods("GET")

	// attach to static assets
	router.PathPrefix("/").Handler(http.FileServer(rice.MustFindBox("assets").HTTPBox()))

	// go serve
	webConf := conf.Web
	go http.ListenAndServe(fmt.Sprintf("%s:%d", webConf.Address, webConf.Port), router)
	fmt.Printf("Started web ui on %s:%d\n", webConf.Address, webConf.Port)

	return nil
}
