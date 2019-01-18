package config

import (
	"fmt"
	"github.com/jessevdk/go-flags"
	"os"
)

type HelpOptions struct {
	Help    bool `short:"h" long:"help" group:"Help options" description:"Show this help message."`
	Version bool `short:"v" long:"version" description:"Print the version of the software."`
}

type AppOptions struct {
	ConfigPath flags.Filename `short:"c" long:"config" description:"Path to Gudgeon configuration file." default:"./gudgeon.yml"`
}

type GudgeonOptions struct {
	// explict app group
	AppOptions AppOptions `group:"Application Options"`

	// emulate help flag with direct support for accessing it
	HelpOptions HelpOptions `group:"Help Options"`
}

func Options(longVersion string) (GudgeonOptions, error) {
	var opts GudgeonOptions
	parser := flags.NewParser(&opts, flags.PassDoubleDash)
	_, err := parser.ParseArgs(os.Args[1:])

	// if version or help we start out the same way
	if opts.HelpOptions.Help || opts.HelpOptions.Version {
		fmt.Printf("[gudgeon] - version: %s\n", longVersion)
		// and then print the rest of the help
		if opts.HelpOptions.Help {
			parser.WriteHelp(os.Stdout)
		}
		// and then we quit here after showing the help
		os.Exit(0)
		// otherwise we just throw back an error
	} else if err != nil {
		return opts, fmt.Errorf("Error parsing commands: %s", err)
	}

	return opts, nil
}
