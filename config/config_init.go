package config

import (
	"strings"
)

func boolPointer(b bool) *bool {
	return &b
}

// encapsulate logic to make it easier to read in this file
func (config *GudgeonConfig) verifyAndInit() ([]string, []error) {
	// initialize maps
	config.resolverMap = make(map[string]*GudgeonResolver, 0)
	config.listMap = make(map[string]*GudgeonList, 0)
	config.consumerMap = make(map[string]*GudgeonConsumer, 0)
	config.groupMap = make(map[string]*GudgeonGroup, 0)

	// collect errors for reporting/combining into one error
	errors := make([]error, 0)
	warnings := make([]string, 0)

	// storage
	if config.Storage == nil {
		config.Storage = &GudgeonStorage{
			RuleStorage: "memory",
		}
	}

	// network verification
	if config.Network == nil {
		config.Network = &GudgeonNetwork{}
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

	return warnings, errors
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
	if metrics.Enabled == nil {
		metrics.Enabled = boolPointer(true)
	}

	if metrics.Persist == nil {
		metrics.Persist = boolPointer(true)
	}

	if "" == metrics.Duration {
		metrics.Duration = "7d"
	}

	if "" == metrics.Interval {
		metrics.Interval = "15s"
	}

	return []string{}, []error{}
}

func (ql *GudgeonQueryLog) verifyAndInit() ([]string, []error) {
	if ql.Enabled == nil {
		ql.Enabled = boolPointer(true)
	}

	if ql.Persist == nil {
		ql.Persist = boolPointer(true)
	}

	if ql.Stdout == nil {
		ql.Stdout = boolPointer(true)
	}

	if "" == ql.Duration {
		ql.Duration = "7d"
	}

	return []string{}, []error{}
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
