package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
	"github.com/chrisruffalo/gudgeon/provider"
)

// pick up version from build process
var Version = "1.0.0"
var GitHash = ""
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
	fmt.Printf("Gudgeon %s\n===============================\n", LongVersion)

	// prepare engine with config options
	engine, err := engine.New(config)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	// create a new provider and start hosting
	provider := provider.NewProvider()
	provider.Host(config, engine)

	// wait for signal
	sig := make(chan os.Signal)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	s := <-sig
	fmt.Printf("Signal (%s) received, stopping\n", s)
}
