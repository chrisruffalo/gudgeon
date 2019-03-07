package config

import (
	"fmt"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/chrisruffalo/gudgeon/util"
)

func boolPointer(b bool) *bool {
	return &b
}

// encapsulate logic to make it easier to read in this file
func (config *GudgeonConfig) verifyAndInit() ([]string, []error) {
	// collect errors for reporting/combining into one error
	errors := make([]error, 0)
	warnings := make([]string, 0)

	// initialize maps
	config.resolverMap = make(map[string]*GudgeonResolver, 0)
	config.listMap = make(map[string]*GudgeonList, 0)
	config.consumerMap = make(map[string]*GudgeonConsumer, 0)
	config.groupMap = make(map[string]*GudgeonGroup, 0)

	// set home dir
	if "" == config.Home {
		usr, err := user.Current()
		if err != nil {
			config.Home = "./.gudgeon"
		} else {
			config.Home = path.Join(usr.HomeDir, ".gudgeon")
		}
		warnings = append(warnings, fmt.Sprintf("No home directory configured, using '%s' for Gudgeon home", config.Home))
	}

	// storage
	if config.Storage == nil {
		config.Storage = &GudgeonStorage{
			RuleStorage:  "memory",
			CacheEnabled: boolPointer(true),
		}
	}
	config.Storage.verifyAndInit()

	// network verification
	if config.Network == nil {
		config.Network = &GudgeonNetwork{
			Interfaces: []*GudgeonInterface{
				&GudgeonInterface{
					IP:   "127.0.0.1",
					Port: 5354,
				},
			},
		}
	}
	warn, err := config.Network.verifyAndInit()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// web defaults and verification
	if config.Web == nil {
		config.Web = &GudgeonWeb{
			Enabled: true,
		}
	}
	warn, err = config.Web.verifyAndInit()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// metrics configuration
	if config.Metrics == nil {
		config.Metrics = &GudgeonMetrics{}
	}
	warn, err = config.Metrics.verifyAndInit()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// query log configuration
	if config.QueryLog == nil {
		config.QueryLog = &GudgeonQueryLog{}
	}
	warn, err = config.QueryLog.verifyAndInit()

	// groups
	warn, err = config.verifyAndInitGroups()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// consumers
	warn, err = config.verifyAndInitConsumers()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// resolvers
	warn, err = config.verifyAndInitResolvers()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	// lists
	warn, err = config.verifyAndInitLists()
	errors = append(errors, err...)
	warnings = append(warnings, warn...)

	return warnings, errors
}

func (storage *GudgeonStorage) verifyAndInit() ([]string, []error) {
	if storage.CacheEnabled == nil {
		storage.CacheEnabled = boolPointer(true)
	}

	return []string{}, []error{}
}

func (web *GudgeonWeb) verifyAndInit() ([]string, []error) {
	if web.Enabled {
		if "" == web.Address {
			web.Address = "127.0.0.1"
		}
		if web.Port < 1 {
			web.Port = 9009
		}
	}

	return []string{}, []error{}
}

func (network *GudgeonNetwork) verifyAndInit() ([]string, []error) {
	// set default values for tcp and udp if nil
	if network.TCP == nil {
		network.TCP = boolPointer(true)
	}
	if network.UDP == nil {
		network.UDP = boolPointer(true)
	}
	if network.Systemd == nil {
		network.Systemd = boolPointer(true)
	}

	// do the same for all configured interfaces
	for _, iface := range network.Interfaces {
		if iface.TCP == nil {
			iface.TCP = network.TCP
		}
		if iface.UDP == nil {
			iface.UDP = network.UDP
		}
	}

	return []string{}, []error{}
}

func (metrics *GudgeonMetrics) verifyAndInit() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	if metrics.Enabled == nil {
		metrics.Enabled = boolPointer(true)
	}

	if metrics.Persist == nil {
		metrics.Persist = boolPointer(true)
	}

	if "" == metrics.Duration {
		metrics.Duration = "7d"
	}
	if _, err := util.ParseDuration(metrics.Duration); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not parse metrics duration: %s, using default (7d)", err))
		metrics.Duration = "7d"
	}

	if "" == metrics.Interval {
		metrics.Interval = "15s"
	}
	if parsed, err := util.ParseDuration(metrics.Interval); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not parse metrics interval: %s, using default (15s)", err))
		metrics.Interval = "15s"
	} else if parsed < time.Second {
		warnings = append(warnings, fmt.Sprintf("A metrics interval less than 1s is probably too short, using default value (15s)"))
		metrics.Interval = "15s"
	} else if parsed > 30*time.Minute {
		warnings = append(warnings, fmt.Sprintf("A metrics interval more than 30 minutes (30m) is fairly low resolution, consider changing this value"))
	}

	return warnings, []error{}
}

func (ql *GudgeonQueryLog) verifyAndInit() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	if ql.Enabled == nil {
		ql.Enabled = boolPointer(true)
	}

	if ql.Persist == nil {
		ql.Persist = boolPointer(true)
	}

	if ql.Stdout == nil {
		ql.Stdout = boolPointer(true)
	}

	if ql.ReverseLookup == nil {
		ql.ReverseLookup = boolPointer(true)
	}

	if ql.MdnsLookup == nil {
		ql.MdnsLookup = boolPointer(true)
	}

	if ql.NetbiosLookup == nil {
		ql.NetbiosLookup = boolPointer(true)
	}

	if "" == ql.Duration {
		ql.Duration = "7d"
	}
	if parsed, err := util.ParseDuration(ql.Duration); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not parse query log duration: %s, using default (7d)", err))
		ql.Duration = "7d"
	} else if parsed < time.Hour {
		warnings = append(warnings, fmt.Sprintf("A query log duration less than 1 hour (1h) is probably too short, using 1h"))
		ql.Duration = "1h"
	}

	if "" == ql.BatchInterval {
		ql.BatchInterval = "1s"
	}
	if parsed, err := util.ParseDuration(ql.BatchInterval); err != nil {
		warnings = append(warnings, fmt.Sprintf("Could not parse query log batch interval: %s, using default", err))
		ql.BatchInterval = "1s"
	} else if parsed > time.Minute {
		warnings = append(warnings, fmt.Sprintf("A batch interval greater than 1m is probably too long, consider changing this value (%s)", ql.BatchInterval))
	} else if parsed < 500*time.Millisecond {
		warnings = append(warnings, fmt.Sprintf("A batch interval less than 500ms is probably too short, using default value (1s)"))
		ql.BatchInterval = "1s"
	}

	if ql.BatchSize < 1 {
		// silently use default in cases where it was probably not specified
		ql.BatchSize = 50
	} else if ql.BatchSize > 1000 {
		warnings = append(warnings, fmt.Sprintf("A batch size greater than 1000 is probably too high, using max value (1000)"))
		ql.BatchSize = 1000
	}

	return warnings, []error{}
}

// verify all the groups at once and set the groupMap
func (config *GudgeonConfig) verifyAndInitGroups() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	// add groups to group map
	for _, group := range config.Groups {
		if group == nil {
			continue
		}
		if "" == group.Name {
			warnings = append(warnings, "A group with no name was found in the configuration, a group with no name will not be used.")
			continue
		}
		group.Name = strings.ToLower(group.Name)

		if _, found := config.groupMap[group.Name]; found {
			warnings = append(warnings, "More than one group was found with the name '%s', group names are case insensitive and must be unique.", group.Name)
			continue
		}
		config.groupMap[group.Name] = group
	}

	// if no default group is found, add it
	if _, found := config.groupMap[defaultString]; !found {
		defaultGroup := &GudgeonGroup{
			Name:      defaultString,
			Tags:      &[]string{defaultString},
			Resolvers: []string{defaultString},
		}
		config.Groups = append(config.Groups, defaultGroup)
		config.groupMap[defaultString] = defaultGroup
	}

	return warnings, []error{}
}

// verify all consumers at once, add a default consumer if needed, and set the group map
func (config *GudgeonConfig) verifyAndInitConsumers() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	for _, consumer := range config.Consumers {
		if consumer == nil {
			continue
		}
		if "" == consumer.Name {
			warnings = append(warnings, "A consumer with no name was found in the configuration, a consumer with no name will not be used.")
			continue
		}
		consumer.Name = strings.ToLower(consumer.Name)

		if _, found := config.consumerMap[consumer.Name]; found {
			warnings = append(warnings, "More than one consumer was found with the name '%s', consumer names are case insensitive and must be unique.", consumer.Name)
			continue
		}
		config.consumerMap[consumer.Name] = consumer
	}

	if _, found := config.consumerMap[defaultString]; !found {
		defaultConsumer := &GudgeonConsumer{
			Name:   defaultString,
			Groups: []string{defaultString},
		}
		config.Consumers = append(config.Consumers, defaultConsumer)
		config.consumerMap[defaultString] = defaultConsumer
	}

	return warnings, []error{}
}

func (config *GudgeonConfig) verifyAndInitResolvers() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	for _, resolver := range config.Resolvers {
		if resolver == nil {
			continue
		}
		if "" == resolver.Name {
			warnings = append(warnings, "A resolver with no name was found in the configuration, a resolver with no name will not be used.")
			continue
		}
		resolver.Name = strings.ToLower(resolver.Name)

		if _, found := config.resolverMap[resolver.Name]; found {
			warnings = append(warnings, "More than one resolver was found with the name '%s', resolver names are case insensitive and must be unique.", resolver.Name)
			continue
		}

		// "condition" default and system resolvers in the event that they were only partially configured
		// we could just leave this alone but it flat won't work without a source
		if systemString == resolver.Name || defaultString == resolver.Name {
			if len(resolver.Sources) == 0 {
				resolver.Sources = []string{systemString}
			}
		}

		config.resolverMap[resolver.Name] = resolver
	}

	// inject "system" and "default" resolvers
	if _, found := config.resolverMap[systemString]; !found {
		systemResolver := &GudgeonResolver{
			Name:    systemString,
			Sources: []string{systemString},
		}
		config.Resolvers = append(config.Resolvers, systemResolver)
		config.resolverMap[systemString] = systemResolver
	}

	// (the default resolver just points to the system resolver if no default resovler is otherwise configured
	if _, found := config.resolverMap[defaultString]; !found {
		defaultResolver := &GudgeonResolver{
			Name:    defaultString,
			Sources: []string{systemString},
		}
		config.Resolvers = append(config.Resolvers, defaultResolver)
		config.resolverMap[defaultString] = defaultResolver
	}

	return warnings, []error{}
}

func (config *GudgeonConfig) verifyAndInitLists() ([]string, []error) {
	// collect warnings
	warnings := make([]string, 0)

	for _, list := range config.Lists {
		if list == nil {
			continue
		}
		if "" == list.Source {
			if "" != list.CanonicalName() {
				warnings = append(warnings, "A list named %s does not have a source in the configuration, a list with no source will not be used.", list.CanonicalName())
			} else {
				warnings = append(warnings, "A list with no source was found in the configuration, a list with no source will not be used.")
			}
			continue
		}
		list.Name = strings.ToLower(list.Name)
		config.listMap[list.CanonicalName()] = list
	}

	return warnings, []error{}
}
