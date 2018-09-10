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
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	// load config
	config, cErr := config.Load(string(opts.AppOptions.ConfigPath))
	if cErr != nil {
		fmt.Printf("%s\n", cErr)
		os.Exit(1)
	}

	// debug print config
	fmt.Printf("Config:\n%s\n", config)

	// start engine with config options
}