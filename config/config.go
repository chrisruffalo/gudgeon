package config

import (
    "errors"
	"fmt"
	"io/ioutil"
	"gopkg.in/yaml.v2"
)

// network: general dns network configuration
type GudgeonNetwork struct {
	// tcp: true when tcp is enabled (default) false otherwise
	TCP bool `yaml:"tcp"`
	// udp: true when udp is enabled (default) false otherwise
	UDP bool `yaml:"udp"`
	// endpoints: list of string endpoints that should have dns
	Endpoints []string `yaml:"string"`
}

// paths: gudgeon paths for filesystem stuff
type GudgeonPaths struct {
	// cache: path to the cache directory used by gudgeon as a temp/working store
	Cache string `yaml:"cache"`
}

// blocklists, blacklists, whitelists: different types of lists for domains that gudgeon will evaluate
type GudgeonList struct {
	// the name of the list (also automatically used as a tag)
	Name string `yaml:"name"`
	// the path to the list, remote paths will be downloaded if possible
	Path string `yaml:"path"`
	// the tags that relate to the list for tag filtering/processing
	Tags []string `yaml:"tags"`
}

type GudgeonRemoteList struct {
	// the name of the list (also automatically used as a tag)
	Name string `yaml:"name"`
	// the path to the list, remote paths will be downloaded if possible
	URL string `yaml:"url"`
	// the tags that relate to the list for tag filtering/processing
	Tags []string `yaml:"tags"`
}

// groups: ties end-users (consumers) to various lists.
type GudgeonGroup struct {
	// name: name of the group
	Name string `yaml:"name"`
	// inherit: list of groups to copy settings from
	Inherit []string `yaml:"inherit"`
	// blacklists: names of blacklists to use
	Blacklists []string `yaml:"blacklists"`
	// whitelists: names of whitelists to use
	Whitelists []string `yaml:"whitelists"`
	// blocklists: names of blocklists to use
	Blocklists []string `yaml:"blocklists"`
	// tags: tags to use for tag-based matching
	Tags []string `yaml:"tags"`
}

// range: an IP range for consumer matching
type GudgeonMatchRange struct {
	From string `yaml:"from"`
	To string `yaml:"to"`
}

type GudgeonMatch struct {
	IP string `yaml:"ip"`
	Range GudgeonMatchRange `yaml:"range"`
	Net string `yaml:"net"`
}

type GundgeonConsumer struct {
	Name string `yaml:"name"`
	Groups []string `yaml:"groups"`
	Matches []GudgeonMatch `yaml:"matches"`
}

type GudgeonConfig struct {

	Network GudgeonNetwork `yaml:"network"`

	Paths GudgeonPaths `yaml:"paths"`

	Blacklists []GudgeonList `yaml:"blacklists"`

	Whitelists []GudgeonList `yaml:"whitelists"`

	Blocklists []GudgeonRemoteList `yaml:"blocklists"`

	Groups []GudgeonGroup `yaml:"groups"`

	Consumers []GundgeonConsumer `yaml:"consumers"`
}

func Load(filename string) (GudgeonConfig, error) {
	// config
	var config GudgeonConfig
	// load bytes from filename
	bytes, err := ioutil.ReadFile(filename)

	// return nil config object without config file so that
	if err != nil {
		return config, errors.New(fmt.Sprintf("Could not load file '%s', error: %s", filename, err))
	}

	// if file is read then unmarshal from data
	yErr := yaml.Unmarshal(bytes, &config)
	if yErr != nil {
		return config, errors.New(fmt.Sprintf("Error unmarshaling file '%s', error: %s", filename, yErr))
	}

	// return configuration
	return config, nil
}