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
	"github.com/chrisruffalo/gudgeon/engine"
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
	Serve(conf *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics, qlog qlog.QLog) error
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

// takes a list of metrics entries and condenses them together so it looks
// pretty much the same when zoomed out but at a lower resolution
func condenseMetrics(metricsEntries []*metrics.MetricsEntry) []*metrics.MetricsEntry {
	// use the time distance between the first and last entry to calculate how "far" it is
	stime := metricsEntries[0].AtTime
	etime := metricsEntries[len(metricsEntries) - 1].AtTime
	distance := int64(etime.Sub(stime).Hours())

	// only apply factor over 2 hours
	if distance > 2 {
		// use the hour distance to calculate the factor
		factor := int(distance)

		// max factor
		if factor > 24 {
			factor = 24
		}

		// make a new list to hold entries with a base capacity
		tempMetricsEntries := make([]*metrics.MetricsEntry, 0, len(metricsEntries) / factor)

		// start index at 0
		sidx := 0

		// chomp through the list in sizes of ${factor}
		for sidx < len(metricsEntries) {
			eidx := sidx + factor
			// constrain to list length
			if eidx >= len(metricsEntries) {
				eidx = len(metricsEntries) - 1
			}

			// base entry in metrics entry
			tEntry := metricsEntries[sidx]

			// add to list before processing
			tempMetricsEntries = append(tempMetricsEntries, tEntry)
			
			// if the end is too close to the start we're done
			if eidx <= sidx + 1 {
				break
			}

			// for each remaining entry in this segment we want to add
			// the interval and the all of the times as well as adjust
			// the "at" time at the end
			for _, fEntry := range metricsEntries[sidx + 1:eidx] {
				tEntry.IntervalSeconds += fEntry.IntervalSeconds
				tEntry.AtTime = fEntry.AtTime
				for k, v := range fEntry.Values {
					if _, found := tEntry.Values[k]; found {
						tEntry.Values[k].Inc(v.Value())
					}
				}
			}

			// average entries out according to the number of entries
			// we want the data to look the same, not be coallated
			for _, v := range tEntry.Values {
				v.Set(v.Value()/int64(len(metricsEntries[sidx:eidx])))
			}

			// increment by factor
			sidx += factor
		}

		// use new metrics entries copy
		return tempMetricsEntries
	}

	return metricsEntries
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

	// filter so that only wanted metrics are shown
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

	// highly experimental but makes the UI much more responsive:
	// condensing reduces the resolution of the metrics so that we can see a longer time scale
	// on the same graph without overloading the web ui
	if condense := c.Query("condense"); len(condense) > 0 && ("true" == condense || "1" == condense) && len(metricsEntries) >= 2 {
		metricsEntries = condenseMetrics(metricsEntries)
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
	c.JSON(http.StatusOK, gin.H {
		"total": resultLen,
		"items": results,
	})
}

func (web *web) GetTestComponents(c *gin.Context) {
	consumers := make([]string, 0, len(web.conf.Consumers))
	for _, c := range web.conf.Consumers {
		if c == nil || c.Name == "" {
			continue
		}
		consumers = append(consumers, c.Name)
	}

	groups := make([]string, 0, len(web.conf.Groups)) 
	for _, g := range web.conf.Groups {
		if g == nil || g.Name == "" {
			continue
		}
		groups = append(groups, g.Name)
	}

	resolvers := make([]string, 0, len(web.conf.Resolvers))
	for _, r := range web.conf.Resolvers {
		if r == nil || r.Name == "" {
			continue
		}
		resolvers = append(resolvers, r.Name)
	}

	c.JSON(http.StatusOK, gin.H {
		"consumers": consumers,
		"groups": groups,
		"resolvers": resolvers,
	})
}

func (web *web) GetTestResult(c *gin.Context) {
	domain := c.Query("domain")
	if len(domain) < 1 {
		c.String(http.StatusNotFound, "Domain must be provided")
	}

	if resolver := c.Query("resolver"); len(resolver) > 0 {

	}

	if consumer := c.Query("consumer"); len(consumer) > 0 {
		
	}

	if group := c.Query("group"); len(group) > 0 {

	}
}


func (web *web) Serve(conf *config.GudgeonConfig, engine engine.Engine, metrics metrics.Metrics, qlog qlog.QLog) error {
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
		api.GET("/test/components", web.GetTestComponents)
		api.GET("/test/query", web.GetTestResult)
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
