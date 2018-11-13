package main

import (
	"fmt"
	"os"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine"
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
	//fmt.Printf("Config:\n%s\n", config)

	// prepare engine with config options
	engine, err := engine.New(config)
	if err != nil {
		fmt.Printf("%s\n", err)
	}

	// start engine
	engine.Start()
}
