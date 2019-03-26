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
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	templateFileExtension = ".tmpl"
)

type web struct {
	conf     *config.GudgeonConfig
	server   *http.Server
	metrics  engine.Metrics
	queryLog engine.QueryLog
}

type Web interface {
	Serve(conf *config.GudgeonConfig, engine engine.Engine) error
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

func condenseMetric(target *engine.MetricsEntry, from *engine.MetricsEntry, counter int) {
	target.AtTime = from.AtTime
	target.IntervalSeconds += from.IntervalSeconds

	for k, v := range target.Values {
		if f, found := from.Values[k]; found {
			current := int64(v.Value() * int64(counter-1))
			v.Set((current + f.Value()) / int64(counter))
		}
	}
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

	keepMetrics := []string{}
	if filterStrings := c.Query("metrics"); len(filterStrings) > 0 {
		keepMetrics = strings.Split(filterStrings, ",")
	}

	condenseMetrics := false
	if condense := c.Query("condense"); len(condense) > 0 && ("true" == condense || "1" == condense) {
		condenseMetrics = true
	}

	firstEntry := true
	entryChan := make(chan *engine.MetricsEntry)

	// hold an entry for condensing into
	var heldEntry *engine.MetricsEntry

	// calculate condensation factor
	stime := *queryStart
	etime := *queryEnd
	distance := int(etime.Sub(stime).Hours())
	factor := distance / 2
	if distance >= 24 {
		distance = distance / 2
	}
	if distance > 48 {
		distance = distance / 2
	}
	if factor >= 512 {
		factor = 512
	}
	condenseCounter := 1

	go web.metrics.QueryStream(entryChan, *queryStart, *queryEnd)

	// start empty array
	c.String(http.StatusOK, "[")
	for me := range entryChan {
		if me == nil {
			continue
		}

		// filter out metrics that are not in the "keep list"
		if len(keepMetrics) > 0 {
			for k, _ := range me.Values {
				if !util.StringIn(k, keepMetrics) {
					delete(me.Values, k)
				}
			}
		}

		// if we are condensing, do condense logic, otherwise just
		// stream output
		if condenseMetrics {
			if heldEntry == nil || condenseCounter < 2 {
				heldEntry = me
			} else {
				condenseMetric(heldEntry, me, condenseCounter)
			}
			condenseCounter++
			me = heldEntry
		}

		if !condenseMetrics || (condenseMetrics && condenseCounter >= factor) {
			if !firstEntry {
				c.String(http.StatusOK, ",")
			}
			firstEntry = false
			c.JSON(http.StatusOK, me)

			// if condensing start over
			if condenseMetrics {
				heldEntry = nil
				condenseCounter = 1
			}
		}
	}

	// write any additional held entires
	if heldEntry != nil {
		if !firstEntry {
			c.String(http.StatusOK, ",")
		}
		c.JSON(http.StatusOK, heldEntry)
	}

	// finish array
	c.String(http.StatusOK, "]")
}

func (web *web) GetQueryLogInfo(c *gin.Context) {
	if web.queryLog == nil {
		c.String(http.StatusNotFound, "Query log not enabled")
		return
	}

	// set default query options
	query := &engine.QueryLogQuery{}
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

	c.JSON(http.StatusOK, gin.H{
		"consumers": consumers,
		"groups":    groups,
		"resolvers": resolvers,
	})
}

func (web *web) GetTop(c *gin.Context) {
	var results []*engine.TopInfo

	limit := 5

	// allow limit setting
	if limitQuery := c.Query("limit"); len(limitQuery) > 0 {
		iLimit, err := strconv.Atoi(limitQuery)
		if err == nil {
			limit = iLimit
		}
	}

	topType := c.Params.ByName("type")
	if topType != "" {
		switch strings.ToLower(topType) {
		case "domains":
			results = web.metrics.TopDomains(limit)
		case "lists":
			results = web.metrics.TopLists(limit)
		case "clients":
			results = web.metrics.TopClients(limit)
		case "rules":
			results = web.metrics.TopRules(limit)
		case "types":
			results = web.metrics.TopQueryTypes(limit)
		}
	}

	c.JSON(http.StatusOK, results)
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

func (web *web) Serve(conf *config.GudgeonConfig, engine engine.Engine) error {
	// set metrics endpoint
	web.metrics = engine.Metrics()
	web.queryLog = engine.QueryLog()
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
		api.GET("/metrics/top/:type", web.GetTop)
		// testing/troubleshoting/diagnostics
		api.GET("/test/components", web.GetTestComponents)
		api.GET("/test/query", web.GetTestResult)
		// attach query log
		api.GET("/query/list", web.GetQueryLogInfo)
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
