package main

import (
    "fmt"
    "os"
    "github.com/chrisruffalo/gudgeon/config"
)

// pick up version from build process
var	Version = "1.0.0"
var GitHash = ""
var LongVersion = Version

func main() {
	// load command options
	opts, err := config.Options(LongVersion)
	if err != nil {
		fmt.Printf("Error parsing command options: %s", err)
		os.Exit(1)
	}

	// load config
	config, cErr := config.Load(string(opts.AppOptions.ConfigPath))
	if cErr != nil {
		fmt.Printf("Error loading configuration file: %s", cErr)
		os.Exit(1)
	}

	// start engine with config options
}