package config

import (
    "errors"
	"fmt"
	"io/ioutil"
	"path"

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
type GudgeonList interface {
	path(cachePath string) string
}

type GudgeonLocalList struct {
	// the name of the list (also automatically used as a tag)
	Name string `yaml:"name"`
	// the tags that relate to the list for tag filtering/processing
	Tags []string `yaml:"tags"`
	// the path to the list, remote paths will be downloaded if possible
	Path string `yaml:"path"`
}

type GudgeonRemoteList struct {
	// the name of the list (also automatically used as a tag)
	Name string `yaml:"name"`
	// the tags that relate to the list for tag filtering/processing
	Tags []string `yaml:"tags"`
	// the path to the list, remote paths will be downloaded if possible
	URL string `yaml:"url"`
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

	Blacklists []GudgeonLocalList `yaml:"blacklists"`

	Whitelists []GudgeonLocalList `yaml:"whitelists"`

	Blocklists []GudgeonRemoteList `yaml:"blocklists"`

	Groups []GudgeonGroup `yaml:"groups"`

	Consumers []GundgeonConsumer `yaml:"consumers"`
}

func (list GudgeonRemoteList) path(cachePath string) string {
	name := list.Name
	return path.Join(cachePath, name + ".list")
}

func (list GudgeonLocalList) path(cachePath string) string {
	return list.Path
}

func (config *GudgeonConfig) PathToList(list GudgeonList) string {
	return list.path(config.Paths.Cache)
}

func Load(filename string) (*GudgeonConfig, error) {
	// config
	var config GudgeonConfig
	// load bytes from filename
	bytes, err := ioutil.ReadFile(filename)

	// return nil config object without config file so that
	if err != nil {
		return &config, errors.New(fmt.Sprintf("Could not load file '%s', error: %s", filename, err))
	}

	// if file is read then unmarshal from data
	yErr := yaml.Unmarshal(bytes, &config)
	if yErr != nil {
		return &config, errors.New(fmt.Sprintf("Error unmarshaling file '%s', error: %s", filename, yErr))
	}

	// return configuration
	return &config, nil
}