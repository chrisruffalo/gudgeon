package web

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	templateFileExtension = ".tmpl"
)

type web struct {
	conf     *config.GudgeonConfig
	server   *http.Server
	metrics  metrics.Metrics
	queryLog qlog.QLog
}

type Web interface {
	Serve(conf *config.GudgeonConfig, metrics metrics.Metrics, qlog qlog.QLog) error
	Stop()
}

func New() Web {
	return &web{}
}

// get metrics counter named in query
func (web *web) GetMetrics(c *gin.Context) {
	if web.metrics == nil {
		c.String(http.StatusNotFound, "Metrics not enabled)")
		return
	}

	lists := make([]map[string]string, 0, len(web.conf.Lists))
	for _, list := range web.conf.Lists {
		if strings.ToLower(list.Type) == "allow" {
			continue
		}

		listEntry := make(map[string]string)
		listEntry["short"] = list.ShortName()
		listEntry["name"] = list.CanonicalName()
		lists = append(lists, listEntry)
	}

	metrics := web.metrics.GetAll()

	if filterStrings := c.Query("metrics"); len(filterStrings) > 0 {
		keepMetrics := strings.Split(filterStrings, ",")
		for k, _ := range metrics {
			if !util.StringIn(k, keepMetrics) {
				delete(metrics, k)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"lists":   lists,
	})
}

func (web *web) QueryMetrics(c *gin.Context) {
	if web.metrics == nil {
		c.String(http.StatusNotFound, "Metrics not enabled)")
		return
	}

	var (
		queryStart *time.Time
		queryEnd   *time.Time
	)

	// look for and convert time (seconds since unix epoch) to local date
	if start := c.Query("start"); len(start) > 0 {
		iStart, err := strconv.ParseInt(start, 10, 64)
		if err != nil {
			iStart = 0
		}
		startTime := time.Unix(iStart, 0)
		queryStart = &startTime
	}

	if end := c.Query("end"); len(end) > 0 {
		iEnd, err := strconv.ParseInt(end, 10, 64)
		if err != nil {
			endTime := time.Unix(iEnd, 0)
			queryEnd = &endTime
		}
	}

	if queryStart == nil {
		startTime := time.Unix(0, 0)
		queryStart = &startTime
	}

	if queryEnd == nil {
		endTime := time.Now()
		queryEnd = &endTime
	}

	// get results
	metricsEntries, err := web.metrics.Query(*queryStart, *queryEnd)
	if err != nil {
		c.String(http.StatusServiceUnavailable, "Error retrieving metrics")
		return
	}

	if filterStrings := c.Query("metrics"); len(filterStrings) > 0 {
		keepMetrics := strings.Split(filterStrings, ",")
		for _, entry := range metricsEntries {
			for k, _ := range entry.Values {
				if !util.StringIn(k, keepMetrics) {
					delete(entry.Values, k)
				}
			}
		}
	}

	// return encoded results
	c.JSON(http.StatusOK, metricsEntries)
}

func (web *web) GetQueryLogInfo(c *gin.Context) {
	if web.queryLog == nil {
		c.String(http.StatusNotFound, "Query log not enabled")
		return
	}

	// set default query options
	query := &qlog.QueryLogQuery{}
	if query.Limit < 1 {
		query.Limit = 100 // default limit to 100 entries
	}

	if limit := c.Query("limit"); len(limit) > 0 {
		if "none" == strings.ToLower(limit) {
			query.Limit = 0
		} else {
			iLimit, err := strconv.Atoi(limit)
			if err == nil {
				query.Limit = iLimit
			}
		}
	}

	if skipped := c.Query("skip"); len(skipped) > 0 {
		if iSkipped, err := strconv.Atoi(skipped); err == nil {
			query.Skip = iSkipped
		}
	}

	if blocked := c.Query("blocked"); len(blocked) > 0 {
		if "true" == strings.ToLower(blocked) {
			boolHolder := true
			query.Blocked = &boolHolder
		} else if "false" == strings.ToLower(blocked) {
			boolHolder := false
			query.Blocked = &boolHolder
		}
	}

	if address := c.Query("address"); len(address) > 0 {
		query.Address = address
	}

	if requestDomain := c.Query("rdomain"); len(requestDomain) > 0 {
		query.RequestDomain = requestDomain
	}

	if clientName := c.Query("clientName"); len(clientName) > 0 {
		query.ClientName = clientName
	}

	if responseText := c.Query("responseText"); len(responseText) > 0 {
		query.ResponseText = responseText
	}

	// look for and convert time (seconds since unix epoch) to local date
	if after := c.Query("after"); len(after) > 0 {
		iAfter, err := strconv.ParseInt(after, 10, 64)
		if err == nil {
			afterTime := time.Unix(iAfter, 0)
			query.After = &afterTime
		}
	}

	if before := c.Query("before"); len(before) > 0 {
		iBefore, err := strconv.ParseInt(before, 10, 64)
		if err == nil {
			beforeTime := time.Unix(iBefore, 0)
			query.Before = &beforeTime
		}
	}

	if sort := c.Query("sortby"); len(sort) > 0 {
		query.SortBy = strings.ToLower(sort)
	}

	if direction := c.Query("direction"); len(direction) > 0 {
		query.Direction = strings.ToUpper(direction)
	}

	results, resultLen := web.queryLog.Query(query)

	// query against query log and return encoded results
	c.JSON(http.StatusOK, gin.H{
		"total": resultLen,
		"items": results,
	})
}

func (web *web) Serve(conf *config.GudgeonConfig, metrics metrics.Metrics, qlog qlog.QLog) error {
	// set metrics endpoint
	web.metrics = metrics
	web.queryLog = qlog
	web.conf = conf

	// create new router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// if no route is matched, attempt to serve static assets
	box := rice.MustFindBox("static").HTTPBox()

	// use static serving when no route is detected
	router.NoRoute(web.ServeStatic(box))

	// attach api
	api := router.Group("/api")
	{
		// metrics api
		api.GET("/metrics/current", web.GetMetrics)
		api.GET("/metrics/query", web.QueryMetrics)
		// attach query log
		api.GET("/log", web.GetQueryLogInfo)
	}

	// go serve
	webConf := conf.Web
	address := fmt.Sprintf("%s:%d", webConf.Address, webConf.Port)
	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}
	web.server = srv
	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("Starting server: %s", err)
		}
	}()

	log.Infof("Started web ui on %s", address)

	return nil
}

func (web *web) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := web.server.Shutdown(ctx); err != nil {
		log.Errorf("Server Shutdown: %s", err)
		return
	}
	ctx.Done()
}
