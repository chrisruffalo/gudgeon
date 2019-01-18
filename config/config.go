package config

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/chrisruffalo/gudgeon/util"
)

var remoteProtocols = []string{"http:", "https:"}
var alphaRegex, _ = regexp.Compile("[^a-zA-Z0-9]+")

type GudgeonTLS struct {
	Enabled bool `yaml:"enabled"`
}

// network interface information
type GudgeonInterface struct {
	// the IP of the interface. The interface 0.0.0.0 means "all"
	IP string `yaml:"ip"`
	// The port to listen on (on the given interface), defaults to 53
	Port int `yaml:"port"`
	// Should this port listen on TCP? (defaults to the value of Network.TCP which defaults to true)
	TCP *bool `yaml:"tcp"`
	// Should this port listen on UDP? (defaults to the value of Network.UDP which defaults to true)
	UDP *bool `yaml:"udp"`
	// TLS settings
	TLS *GudgeonTLS `yaml:"tls"`
}

// network: general dns network configuration
type GudgeonNetwork struct {
	// Global TLS settings
	TLS *GudgeonTLS `yaml:"tls"`
	// tcp: true when the default for all interfaces is to use tcp
	TCP *bool `yaml:"tcp"`
	// udp: true when the default for all interfaces is to use udp
	UDP *bool `yaml:"udp"`
	// systemd: also accept listeners request from systemd
	Systemd *bool `yaml:"systemd"`
	// endpoints: list of string endpoints that should have dns
	Interfaces []*GudgeonInterface `yaml:"interfaces"`
}

// a resolver is composed of a list of sources to get DNS information from
type GudgeonResolver struct {
	// name of the resolver
	Name string `yaml:"name"`
	// domains to operate on
	Domains []string `yaml:"domains"`
	// domains to skip
	SkipDomains []string `yaml:"skip"`
	// search domains, will retry resolution using these subdomains if the domain is not found
	Search []string `yaml:"search"`
	// manual hosts
	Hosts []string `yaml:"hosts"`
	// sources (described via string)
	Sources []string `yaml:"sources"`
}

// blocklists, blacklists, whitelists: different types of lists for domains that gudgeon will evaluate
type GudgeonList struct {
	// the name of the list
	Name string `yaml:"name"`
	// the type of the list, requires "allow" or "block", defaults to "block"
	Type string `yaml:"type"`
	// the tags that relate to the list for tag filtering/processing
	Tags []string `yaml:"tags"`
	// the path to the list, remote paths will be downloaded if possible
	Source string `yaml:"src"`
}

// groups: ties end-users (consumers) to various lists.
type GudgeonGroup struct {
	// name: name of the group
	Name string `yaml:"name"`
	// inherit: list of groups to copy settings from
	Inherit []string `yaml:"inherit"`
	// resolvers: resolvers to use for this group
	Resolvers []string `yaml:"resolvers"`
	// lists: names of blocklists to apply
	Lists []string `yaml:"lists"`
	// tags: tags to use for tag-based matching
	Tags []string `yaml:"tags"`
}

// range: an IP range for consumer matching
type GudgeonMatchRange struct {
	Start string `yaml:"start"`
	End   string `yaml:"end"`
}

type GudgeonMatch struct {
	IP    string             `yaml:"ip"`
	Range *GudgeonMatchRange `yaml:"range"`
	Net   string             `yaml:"net"`
}

type GundgeonConsumer struct {
	Name    string          `yaml:"name"`
	Groups  []string        `yaml:"groups"`
	Matches []*GudgeonMatch `yaml:"matches"`
}

type GudgeonWeb struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type GudgeonConfig struct {
	Home string `yaml:"home"`

	Network *GudgeonNetwork `yaml:"network"`

	Web *GudgeonWeb `yaml:"web"`

	Resolvers []*GudgeonResolver `yaml:"resolvers"`

	Lists []*GudgeonList `yaml:"lists"`

	Groups []*GudgeonGroup `yaml:"groups"`

	Consumers []*GundgeonConsumer `yaml:"consumers"`
}

type GudgeonRoot struct {
	Config *GudgeonConfig `yaml:"gudgeon"`
}

// simple function to get source as name if name is missing
func (list *GudgeonList) CanonicalName() string {
	if "" == list.Name {
		return list.Source
	}
	return list.Name
}

func (list *GudgeonList) ShortName() string {
	name := strings.ToLower(list.CanonicalName())
	return alphaRegex.ReplaceAllString(name, "")
}

func (list *GudgeonList) IsRemote() bool {
	return list != nil && "" != list.Source && util.StartsWithAny(list.Source, remoteProtocols)
}

func (list *GudgeonList) path(cachePath string) string {
	source := list.Source
	if list.IsRemote() {
		return path.Join(cachePath, list.Name+".list")
	}
	return source
}

func (config *GudgeonConfig) PathToList(list *GudgeonList) string {
	if list == nil {
		return ""
	}
	return list.path(config.CacheRoot())
}

func (config *GudgeonConfig) SessionRoot() string {
	return path.Join(config.Home, "sessions")
}

func (config *GudgeonConfig) CacheRoot() string {
	return path.Join(config.Home, "cache")
}

func Load(filename string) (*GudgeonConfig, error) {
	// config from config root
	root := new(GudgeonRoot)

	// load bytes from filename
	bytes, err := ioutil.ReadFile(filename)

	// return nil config object without config file (propagate error)
	if err != nil {
		return nil, fmt.Errorf("Could not load file '%s', error: %s", filename, err)
	}

	// if file is read then unmarshal from data
	yErr := yaml.Unmarshal(bytes, root)
	if yErr != nil {
		return nil, fmt.Errorf("Error unmarshaling file '%s', error: %s", filename, yErr)
	}

	// get config
	config := root.Config
	verifyErr := config.verifyAndInit()
	if verifyErr != nil {
		return nil, verifyErr
	}

	// return configuration
	return config, nil
}
