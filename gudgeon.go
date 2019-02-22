package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/metrics"
	"github.com/chrisruffalo/gudgeon/provider"
	gqlog "github.com/chrisruffalo/gudgeon/qlog"
	"github.com/chrisruffalo/gudgeon/util"
	"github.com/chrisruffalo/gudgeon/web"
)

// pick up version from build process, but use these defaults
var Version = "v0.3.X"
var GitHash = "0000000"
var LongVersion = Version

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
	fmt.Printf("===============================\nGudgeon %s\n===============================\n", LongVersion)

	// load config
	config, err := config.Load(string(opts.AppOptions.ConfigPath))
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// clean out session directory
	if "" != config.SessionRoot() {
		util.ClearDirectory(config.SessionRoot())
	}

	// create metrics
	var mets metrics.Metrics
	if *config.Metrics.Enabled {
		mets = metrics.New(config)
	}

	// create query log
	var qlog gqlog.QLog
	if *config.QueryLog.Enabled {
		qlog, err = gqlog.New(config)
		if err != nil {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
	}

	// prepare engine with config options
	engine, err := engine.New(config, mets)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// create a new provider and start hosting
	provider := provider.NewProvider()
	provider.Host(config, engine, mets, qlog)

	// open web ui if web enabled
	if config.Web.Enabled {
		web := web.New()
		web.Serve(config, mets, qlog)
	}

	// try and print out error if we caught one during startup
	if recovery := recover(); recovery != nil {
		fmt.Printf("An error occured while starting Gudgeon: %s\n", recovery)
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
}
