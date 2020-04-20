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
	"github.com/miekg/dns"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/resolver"
)

type web struct {
	conf   *config.GudgeonConfig
	server *http.Server

	engine engine.Engine
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
	if web.engine.Metrics() == nil {
		c.String(http.StatusNotFound, "Metrics not enabled)")
		return
	}

	var metrics map[string]*engine.Metric
	if filterStrings := c.Query("metrics"); len(filterStrings) > 0 {
		keepMetrics := strings.Split(filterStrings, ",")
		metrics = make(map[string]*engine.Metric, len(keepMetrics))
		for _, key := range keepMetrics {
			metrics[key] = web.engine.Metrics().Get(key)
		}
	} else {
		metrics = *web.engine.Metrics().GetAll()
	}

	c.JSON(http.StatusOK, &gin.H{
		"metrics": metrics,
		"lists":   web.engine.Lists(),
	})
}

func (web *web) QueryMetrics(c *gin.Context) {
	if web.engine.Metrics() == nil {
		c.String(http.StatusNotFound, "Metrics not enabled")
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
			iStart = time.Now().Unix() - int64(web.engine.Metrics().Duration().Seconds())
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

	// status is ok
	c.String(http.StatusOK, "[")

	// get metrics filter string
	chosenMetrics := c.Query("metrics")

	// parse step size (which is really a ratio to reduce/average metrics)
	stepSize := 1
	stepSizeString := c.Query("step")
	if len(stepSizeString) > 0 {
		parsedStepSize, err := strconv.ParseInt(stepSizeString, 10, 64)
		if err == nil {
			stepSize = int(parsedStepSize)
		}
	}
	// keep narrow range of sizes
	if stepSize < 1 {
		stepSize = 1
	}
	if stepSize > 20 {
		stepSize = 20
	}

	// create query options
	options := engine.QueryOptions{
		chosenMetrics,
		stepSize,
	}

	// when on the first entry the output is slightly different
	firstEntry := true
	// call and accumulate responses directly to stream
	err := web.engine.Metrics().QueryFunc(func(entry *engine.MetricsEntry) {
		if !firstEntry {
			c.String(http.StatusOK, ",")
		}
		firstEntry = false
		// custom marshalling that might be done better another way but
		// it prevents the core metrics values from being unmarshalled and then
		// remarshalled within the same request
		c.String(http.StatusOK, "{ \"AtTime\": ")
		c.JSON(http.StatusOK, entry.AtTime)
		c.String(http.StatusOK, ", \"FromTime\": ")
		c.JSON(http.StatusOK, entry.FromTime)
		c.String(http.StatusOK, ", \"IntervalSeconds\": ")
		c.JSON(http.StatusOK, entry.IntervalSeconds)
		c.String(http.StatusOK, ", \"Values\": ")
		_, _ = c.Writer.Write(entry.JsonBytes)
		c.String(http.StatusOK, "}")
	}, options, false, *queryStart, *queryEnd)

	if err != nil {
		c.String(http.StatusInternalServerError, "Could not fetch metrics")
		log.Errorf("Fetching metrics: %s", err)
		return
	}

	// finish array
	c.String(http.StatusOK, "]")
}

func (web *web) GetQueryLogInfo(c *gin.Context) {
	if web.engine.QueryLog() == nil {
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

	if matchRule := c.Query("matchRule"); len(matchRule) > 0 {
		query.MatchRule = matchRule
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

	c.String(http.StatusOK, "{")
	firstRecord := true
	web.engine.QueryLog().QueryFunc(query, func(count uint64, info *engine.InfoRecord) {
		// skip nil records
		if info == nil {
			return
		}

		if firstRecord {
			c.String(http.StatusOK, "\"total\": %d", count)
			c.String(http.StatusOK, ", \"items\": [")
		} else {
			c.String(http.StatusOK, ", ")
		}
		firstRecord = false
		c.JSON(http.StatusOK, info)
	})
	// firstRecord is still true which means no results
	if firstRecord {
		c.String(http.StatusOK, "\"total\": 0")
		c.String(http.StatusOK, ", \"items\": [")
	}
	c.String(http.StatusOK, "]}")
}

func (web *web) GetTestComponents(c *gin.Context) {
	// calling it "consumer" here is because we can only
	// accept singular consumers at the api endpoint and
	// this is consistent with that and the naming of the
	// option in the drop down list
	c.JSON(http.StatusOK, &gin.H{
		"consumer":  web.engine.Consumers(),
		"groups":    web.engine.Groups(),
		"resolvers": web.engine.Resolvers(),
	})
}

func (web *web) GetTop(c *gin.Context) {
	if web.engine.Metrics() == nil || !(*web.conf.Metrics.Detailed) {
		c.String(http.StatusNotFound, "Detailed Metrics not enabled)")
		return
	}

	var results []*engine.TopInfo

	// limit to 5 by default
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
			results = web.engine.Metrics().TopDomains(limit)
		case "lists":
			results = web.engine.Metrics().TopLists(limit)
		case "clients":
			results = web.engine.Metrics().TopClients(limit)
		case "rules":
			results = web.engine.Metrics().TopRules(limit)
		case "types":
			results = web.engine.Metrics().TopQueryTypes(limit)
		}
	}

	c.JSON(http.StatusOK, results)
}

func (web *web) GetTestResult(c *gin.Context) {
	domain := c.Query("domain")
	if len(domain) < 1 {
		c.String(http.StatusNotFound, "Domain must be provided")
	}
	domain = dns.Fqdn(domain)

	// get query type
	qtype := strings.ToUpper(c.Query("qtype"))
	if len(qtype) < 1 {
		c.String(http.StatusNotFound, "Query type must be provided")
	} else {
		if _, found := dns.StringToType[qtype]; !found {
			c.String(http.StatusNotFound, "Query type must be a valid DNS query type")
		}
	}

	// create a new question from domain given
	question := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Authoritative:     true,
			AuthenticatedData: true,
			RecursionDesired:  true,
			Opcode:            dns.OpcodeQuery,
		},
	}
	question.Question = []dns.Question{{Name: domain, Qclass: dns.ClassINET, Qtype: dns.StringToType[qtype]}}

	var (
		response *dns.Msg
		result   *resolver.ResolutionResult
	)

	rCon := resolver.DefaultRequestContext()
	rCon.Protocol = "tcp"

	if consumer := c.Query("consumer"); len(consumer) > 0 {
		response, _, result = web.engine.HandleWithConsumerName(consumer, rCon, question)
	} else if groups := c.Query("groups"); len(groups) > 0 {
		response, _, result = web.engine.HandleWithGroups([]string{groups}, rCon, question)
	} else if resolvers := c.Query("resolvers"); len(resolvers) > 0 {
		response, _, result = web.engine.HandleWithResolvers([]string{resolvers}, rCon, question)
	}

	var responseText string
	if response != nil {
		responseText = response.String()
	}

	c.JSON(http.StatusOK, &gin.H{
		"response": response,
		"result":   result,
		"text":     responseText,
	})
}

func (web *web) Serve(conf *config.GudgeonConfig, engine engine.Engine) error {
	// set metrics endpoint
	web.engine = engine
	web.conf = conf

	// create new router
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// if no route is matched, attempt to serve static assets
	box := rice.MustFindBox("static")

	// use static serving when no route is detected
	router.NoRoute(web.ServeStatic(box))

	// attach api
	api := router.Group("/api")
	{
		// metrics api
		api.GET("/metrics/current", web.GetMetrics)
		api.GET("/metrics/query", web.QueryMetrics)
		api.GET("/metrics/top/:type", web.GetTop)
		// testing/troubleshooting/diagnostics
		api.GET("/test/components", web.GetTestComponents)
		api.GET("/test/query", web.GetTestResult)
		// attach query log
		api.GET("/query/list", web.GetQueryLogInfo)
	}

	// go serve
	webConf := conf.Web
	address := fmt.Sprintf("%s:%d", webConf.Address, webConf.Port)
	web.server = &http.Server{
		Addr:    address,
		Handler: router,
	}
	go func() {
		// service connections
		if err := web.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
