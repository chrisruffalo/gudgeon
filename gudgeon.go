package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/gops/agent"
	"github.com/pkg/profile"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/provider"
	gqlog "github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/util"
	"github.com/chrisruffalo/gudgeon/web"
)

// default divider
var divider = "==============================="

// pick up version from build process, but use these defaults
var Version = "v0.3.X"
var GitHash = "0000000"
var LongVersion = Version

type Gudgeon struct {
	config   *config.GudgeonConfig
	engine   engine.Engine
	provider provider.Provider
	qlog     gqlog.QLog
	metrics  metrics.Metrics
	web      web.Web
}

func NewGudgeon(config *config.GudgeonConfig) *Gudgeon {
	return &Gudgeon{
		config: config,
	}
}

func (gudgeon *Gudgeon) Start() error {
	// get config
	config := gudgeon.config

	// error
	var err error

	// clean out session directory
	if "" != config.SessionRoot() {
		util.ClearDirectory(config.SessionRoot())
	}

	// create metrics
	var mets metrics.Metrics
	if *config.Metrics.Enabled {
		mets = metrics.New(config)
		gudgeon.metrics = mets
	}

	// create query log
	var qlog gqlog.QLog
	if *config.QueryLog.Enabled {
		qlog, err = gqlog.New(config)
		if err != nil {
			return err
		}
		gudgeon.qlog = qlog
	}

	// prepare engine with config options
	engine, err := engine.New(config, mets)
	if err != nil {
		return err
	}
	gudgeon.engine = engine

	// create a new provider and start hosting
	provider := provider.NewProvider()
	provider.Host(config, engine, mets, qlog)
	gudgeon.provider = provider

	// open web ui if web enabled
	if config.Web.Enabled {
		web := web.New()
		web.Serve(config, mets, qlog)
		gudgeon.web = web
	}

	// try and print out error if we caught one during startup
	if recovery := recover(); recovery != nil {
		return fmt.Errorf("unrecoverable error: %s\n", recovery)
	}

	return nil
}

func (gudgeon *Gudgeon) Shutdown() {
	// stop providers
	if gudgeon.provider != nil {
		gudgeon.provider.Shutdown()
	}

	// stop web

	// stop metrics
	if gudgeon.metrics != nil {
		gudgeon.metrics.Stop()
	}

	// stop query log
	if gudgeon.qlog != nil {
		gudgeon.qlog.Stop()
	}
}

func main() {
	// add git hash to long version if available
	if "" != GitHash {
		LongVersion = Version + "@git" + GitHash
	}

	// load command options
	opts, err := config.Options(LongVersion)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// debug print config
	fmt.Printf("%s\nGudgeon %s\n%s\n", divider, LongVersion, divider)

	// start profiling if enabled
	if opts.DebugOptions.Profile {
		fmt.Printf("Starting profiling...\n")
		// start profile
		defer profile.Start().Stop()
		// start agent
		err := agent.Listen(agent.Options{})
		if err != nil {
			fmt.Printf("Could not starting GOPS profilling agent: %s\n", err)
		}
	}

	// load config
	config, err := config.Load(string(opts.AppOptions.ConfigPath))
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// create new Gudgeon instance
	instance := NewGudgeon(config)

	// start new instance
	err = instance.Start()
	if err != nil {
		fmt.Printf("Error starting Gudgeon: %s\n", err)
		os.Exit(1)
	}

	// wait for signal
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig

	// clean out session directory
	if "" != config.SessionRoot() {
		util.ClearDirectory(config.SessionRoot())
	}

	fmt.Printf("Signal (%s) received, stopping\n", s)
	// stop gudgeon, hopefully gracefully
	instance.Shutdown()
}
