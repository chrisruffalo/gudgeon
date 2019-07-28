package config

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/chrisruffalo/gudgeon/util"
)

type ListType uint8
type ListString string

const (
	// the constant that means ALLOW after pasring "allow" or "block"
	ALLOW = ListType(1)
	// the constant that means BLOCK after pasring "allow" or "block"
	BLOCK = ListType(0)

	// the string that represents "allow", all other results are treated as "block"
	ALLOWSTRING = ListString("allow")
	BLOCKSTRING = ListString("block")

	defaultString = "default"
	systemString  = "system"
)

var remoteProtocols = []string{"http:", "https:"}

type GudgeonTLS struct {
	Enabled bool `yaml:"enabled"`
}

type GudgeonSystemd struct {
	// should we accept ports from systemd?
	Enabled *bool `yaml:"enabled"`
	// ports that will be interpreted as "dns" ports
	DnsPorts *[]uint32 `yaml:"dns"` // default 53
	// ports that will be interpreted as "http" ports
	HttpPorts *[]uint32 `yaml:"http"` // default 80, 8080
}

type GudgeonDatabase struct {
	// how often the transient query log table is flushed to metrics and indexed query log tables
	Flush string `yaml:"flush"`
}

type GudgeonQueryLog struct {
	// controls if the entire feature is enabled/disabled
	Enabled *bool `yaml:"enabled"`
	// enables persisting queries to the log (to make them searchable)
	Persist *bool `yaml:"persist"`
	// how long to keep the queries for
	Duration string `yaml:"duration"`
	// if we should also log to stdout
	Stdout *bool `yaml:"stdout"`
	// if we should log to a file, the path to that file (does not rotate automatically)
	File string `yaml:"file"`
	// reverse lookup using query engine
	ReverseLookup *bool `yaml:"lookup"`
	// add mdns/zeroconf/bonjour capability to lookup
	MdnsLookup *bool `yaml:"mdns"`
	// add netbios capability to lookup
	NetbiosLookup *bool `yaml:"netbios"`
}

type GudgeonMetrics struct {
	// controls if the entire feature is enabled/disabled
	Enabled *bool `yaml:"enabled"`
	// enables metrics persisting to the db
	Persist *bool `yaml:"persist"`
	// enables detailed stats per client, domain, etc
	Detailed *bool `yaml:"detailed"`
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
	// endpoints: list of string endpoints that should have dns
	Interfaces []*GudgeonInterface `yaml:"interfaces"`
}

// provides more configuration options and details for sources beyond the simple source specification
type GudgeonSource struct {
	// name that would be in the source list for a resolver
	Name string `yaml:"name"`
	// specs of children resolvers (same as a simple source spec)
	Specs []string `yaml:"spec"`
	// should the entries in the spec list be load balanced (default: false)
	LoadBalance bool `yaml:"load_balance"`
	// source specific options to allow further configuration of sources
	Options map[string]interface{} `yaml:"options"`
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

// GudgeonList different types of lists for domains that gudgeon will evaluate (and if they explicitly allow or block the matched entries)
type GudgeonList struct {
	// the name of the list
	Name      string `yaml:"name"`
	shortName string `yaml:"-"`
	// the type of the list, requires "allow" or "block", defaults to "block"
	Type       string   `yaml:"type"`
	parsedType ListType `yaml:"-"`
	// should items in the list be interpreted as **regex only**
	Regex *bool `yaml:"regex"`
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
	// in case we miss verify/init and initial set up of list (like with test cases)
	if "" == list.shortName {
		return list.Name
	}
	return list.shortName
}

func (list *GudgeonList) IsRemote() bool {
	return list != nil && "" != list.Source && util.StartsWithAny(list.Source, remoteProtocols)
}

func (list *GudgeonList) path(cachePath string) string {
	// if the list is remote use the internal cache path
	if list.IsRemote() {
		return path.Join(cachePath, list.ShortName()+".list")
	}
	// if not remote try and find along path
	source := list.Source
	if absPath, err := filepath.Abs(source); err == nil {
		source = absPath
	}
	if evalPath, err := filepath.EvalSymlinks(source); err == nil {
		source = evalPath
	}
	return source
}

func (list *GudgeonList) ParsedType() ListType {
	return list.parsedType
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
	Systemd   *GudgeonSystemd    `yaml:"systemd"`
	Storage   *GudgeonStorage    `yaml:"storage"`
	Database  *GudgeonDatabase   `yaml:"database"`
	Metrics   *GudgeonMetrics    `yaml:"metrics"`
	QueryLog  *GudgeonQueryLog   `yaml:"query_log"`
	Network   *GudgeonNetwork    `yaml:"network"`
	Web       *GudgeonWeb        `yaml:"web"`
	Sources   []*GudgeonSource   `yaml:"sources"`
	Resolvers []*GudgeonResolver `yaml:"resolvers"`
	Lists     []*GudgeonList     `yaml:"lists"`
	Groups    []*GudgeonGroup    `yaml:"groups"`
	Consumers []*GudgeonConsumer `yaml:"consumers"`

	// private values
	sourceMap   map[string]*GudgeonSource
	resolverMap map[string]*GudgeonResolver
	listMap     map[string]*GudgeonList
	groupMap    map[string]*GudgeonGroup
	consumerMap map[string]*GudgeonConsumer
}

func (config *GudgeonConfig) GetResolver(name string) *GudgeonResolver {
	if value, found := config.resolverMap[name]; found {
		return value
	}
	return nil
}

func (config *GudgeonConfig) GetList(name string) *GudgeonList {
	if value, found := config.listMap[name]; found {
		return value
	}
	return nil
}

func (config *GudgeonConfig) GetGroup(name string) *GudgeonGroup {
	if value, found := config.groupMap[name]; found {
		return value
	}
	return nil
}

func (config *GudgeonConfig) GetConsumer(name string) *GudgeonConsumer {
	if value, found := config.consumerMap[name]; found {
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

	if config == nil {
		return nil, []string{}, fmt.Errorf("Loaded a nil configuration")
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
