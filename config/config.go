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

const (
	defaultString = "default"
	systemString  = "system"
)

var remoteProtocols = []string{"http:", "https:"}
var alphaRegex, _ = regexp.Compile("[^a-zA-Z0-9]+")

type GudgeonTLS struct {
	Enabled bool `yaml:"enabled"`
}

type GudgeonQueryLog struct {
	// controls if the entire feature is enabled/disabled
	Enabled       *bool  `yaml:"enabled"`
	// enables persisting queries to the log (to make them searchable)
	Persist       *bool  `yaml:"persist"`
	// how long to keep the queries for
	Duration      string `yaml:"duration"`
	// if we should also log to stdout
	Stdout        *bool  `yaml:"stdout"`
	// if we should log to a file, the path to that file (does not rotate automatically)
	File          string `yaml:"file"`
	// how often to flush outstanding transactions/queries
	BatchInterval string `yaml:"interval"`
	// reverse lookup using query engine
	ReverseLookup *bool  `yaml:"lookup"`
	// add mdns/zeroconf/bonjour capability to lookup
	MdnsLookup    *bool  `yaml:"mdns"`
	// add netbios capability to lookup
	NetbiosLookup *bool  `yaml:"netbios"`
}

type GudgeonMetrics struct {
	// controls if the entire feature is enabled/disabled
	Enabled  *bool  `yaml:"enabled"`
	// enables metrics persisting to the db
	Persist  *bool  `yaml:"persist"`
	// enables detailed stats per client, domain, etc
	Detailed *bool  `yaml:"detailed"`
	// how long to keep records
	Duration string `yaml:"duration"`
	// how often to record metrics
	Interval string `yaml:"interval"`
}

// GudgeonStorage defines the different storage types for persistent/session data
type GudgeonStorage struct {
	// rule storage is used by the rule storage engine to decide which implementation to use
	// values are like 'memory', 'sqlite', 'hash32', etc
	RuleStorage string `yaml:"rules"`
	// you can enable/disable the cache here, default is to enable
	CacheEnabled *bool `yaml:"cache"`
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
	Tags *[]string `yaml:"tags"`
	// the path to the list, remote paths will be downloaded if possible
	Source string `yaml:"src"`
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
	return alphaRegex.ReplaceAllString(name, "_")
}

func (list *GudgeonList) IsRemote() bool {
	return list != nil && "" != list.Source && util.StartsWithAny(list.Source, remoteProtocols)
}

func (list *GudgeonList) path(cachePath string) string {
	source := list.Source
	if list.IsRemote() {
		return path.Join(cachePath, list.ShortName()+".list")
	}
	return source
}

func (list *GudgeonList) SafeTags() []string {
	if list.Tags == nil {
		return []string{"default"}
	} else if len(*list.Tags) == 0 {
		return []string{}
	}
	return *(list.Tags)
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
	Tags *[]string `yaml:"tags"`
}

func (list *GudgeonGroup) SafeTags() []string {
	if list.Tags == nil {
		return []string{"default"}
	} else if len(*list.Tags) == 0 {
		return []string{}
	}
	return *(list.Tags)
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

type GudgeonConsumer struct {
	Name    string          `yaml:"name"`
	Block   bool            `yaml:"block"`
	Groups  []string        `yaml:"groups"`
	Matches []*GudgeonMatch `yaml:"matches"`
}

type GudgeonWeb struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type GudgeonConfig struct {
	Home      string             `yaml:"home"`
	Storage   *GudgeonStorage    `yaml:"storage"`
	Metrics   *GudgeonMetrics    `yaml:"metrics"`
	QueryLog  *GudgeonQueryLog   `yaml:"query_log"`
	Network   *GudgeonNetwork    `yaml:"network"`
	Web       *GudgeonWeb        `yaml:"web"`
	Resolvers []*GudgeonResolver `yaml:"resolvers"`
	Lists     []*GudgeonList     `yaml:"lists"`
	Groups    []*GudgeonGroup    `yaml:"groups"`
	Consumers []*GudgeonConsumer `yaml:"consumers"`

	// private values
	resolverMap map[string]*GudgeonResolver
	listMap     map[string]*GudgeonList
	groupMap    map[string]*GudgeonGroup
	consumerMap map[string]*GudgeonConsumer
}

func (conf *GudgeonConfig) GetResolver(name string) *GudgeonResolver {
	if value, found := conf.resolverMap[name]; found {
		return value
	}
	return nil
}

func (conf *GudgeonConfig) GetList(name string) *GudgeonList {
	if value, found := conf.listMap[name]; found {
		return value
	}
	return nil
}

func (conf *GudgeonConfig) GetGroup(name string) *GudgeonGroup {
	if value, found := conf.groupMap[name]; found {
		return value
	}
	return nil
}

func (conf *GudgeonConfig) GetConsumer(name string) *GudgeonConsumer {
	if value, found := conf.consumerMap[name]; found {
		return value
	}
	return nil
}

type GudgeonRoot struct {
	Config *GudgeonConfig `yaml:"gudgeon"`
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

func (config *GudgeonConfig) DataRoot() string {
	return path.Join(config.Home, "data")
}

func Load(filename string) (*GudgeonConfig, []string, error) {
	var config *GudgeonConfig

	warnings := make([]string, 0)

	if "" == filename {
		warnings = append(warnings, "No file provided with '-c' flag, using configuration defaults...")
		config = &GudgeonConfig{}
	} else {
		// config from config root
		root := new(GudgeonRoot)

		// load bytes from filename
		bytes, err := ioutil.ReadFile(filename)

		// return nil config object without config file (propagate error)
		if err != nil {
			return nil, []string{}, fmt.Errorf("Load file '%s', error: %s", filename, err)
		}

		// if file is read then unmarshal from data
		yErr := yaml.Unmarshal(bytes, root)
		if yErr != nil {
			return nil, []string{}, fmt.Errorf("Unmarshaling file '%s', error: %s", filename, yErr)
		}

		// get config
		config = root.Config
	}

	// get warnings and errors
	addWarnings, errors := config.verifyAndInit()
	warnings = append(warnings, addWarnings...)

	// bail and return errors
	if len(errors) > 0 {
		errorStrings := make([]string, len(errors))
		for idx, err := range errors {
			errorStrings[idx] = fmt.Sprintf("%s\n", err)
		}
		return nil, []string{}, fmt.Errorf("Errors loading the configuration file:\n%s", strings.Join(errorStrings, ""))
	}

	// return configuration
	return config, warnings, nil
}
