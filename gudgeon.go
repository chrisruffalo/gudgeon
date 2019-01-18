package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/provider"
	"github.com/chrisruffalo/gudgeon/util"
	"github.com/chrisruffalo/gudgeon/web"
)

// pick up version from build process
var Version = "1.0.0"
var GitHash = "000000"
var LongVersion = Version + "@git" + GitHash

func main() {
	// load command options
	opts, err := config.Options(LongVersion)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// load config
	config, err := config.Load(string(opts.AppOptions.ConfigPath))
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// debug print config
	fmt.Printf("===============================\nGudgeon %s\n===============================\n", LongVersion)

	// clean out session directory
	if "" != config.SessionRoot() {
		util.ClearDirectory(config.SessionRoot())
	}

	// prepare engine with config options
	engine, err := engine.New(config)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// create a new provider and start hosting
	provider := provider.NewProvider()
	provider.Host(config, engine)

	// open web ui if web enabled
	if config.Web.Enabled {
		web := web.New()
		web.Serve(config)
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
