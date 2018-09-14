package engine

import (
	"bufio"
	"fmt"
	"os"

	"github.com/willf/bloom"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/downloader"
	"github.com/chrisruffalo/gudgeon/util"
)

const (
	bloomFilterAcceptableError = float64(0.0005) // rate of acceptable error
)

type activeGroup struct {
	configGroup *config.GudgeonGroup
	whiteFilter *bloom.BloomFilter
	blackFilter *bloom.BloomFilter
	blockFilter *bloom.BloomFilter

	whitelists []*config.GudgeonList
	blacklists []*config.GudgeonList
	blocklists []*config.GudgeonList
}

type activeConsumer struct {
	consumer *config.GundgeonConsumer
	groups []*activeGroup
}

type engine struct {
	config *config.GudgeonConfig
	consumers []*activeConsumer
	defaultGroup *activeGroup
}

type Engine interface {
	IsDomainBlocked(consumer string, domain string) bool
	Start() error
}

func shouldAssignList(listNames []string, listTags []string, lists []*config.GudgeonList) []*config.GudgeonList {
	// empty list
	should := []*config.GudgeonList{}

	// check names
	for _, list := range lists {
		if util.StringIn(list.Name, listNames) {
			should = append(should, list)
			continue
		}

		for _, tag := range list.Tags {
			if util.StringIn(tag, listTags) {
				should = append(should, list)
				break
			}
		}
	}

	// return the list of names
	return should
}

func createBloomFilter(config *config.GudgeonConfig, lists []*config.GudgeonList) *bloom.BloomFilter {
	// count lines
	totalLines := uint(0)
	for _, list := range lists {
		lines, err := util.LinesInFile(config.PathToList(list))
		if err != nil {
			continue
		}
		totalLines += lines
	}

	// create filter with acceptable error
	filter := bloom.NewWithEstimates(totalLines, bloomFilterAcceptableError)

	// load lines
	for _, list := range lists {
		reader, err := os.Open(config.PathToList(list))
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			// only add to bloom filter if rule is not complex (ie: straight block)
			if !IsComplexRule(text) {
				filter.AddString(scanner.Text())	
			}				
		}
	}

	return filter
}

func New(conf *config.GudgeonConfig) (Engine, error) {
	// make required paths
	os.MkdirAll(conf.Paths.Cache, os.ModePerm)

	// create return object
	engine := new(engine)
	engine.config = conf

	lists := []*config.GudgeonList{}
	lists = append(lists, conf.Whitelists...)
	lists = append(lists, conf.Blacklists...)
	lists = append(lists, conf.Blocklists...)

	// load lists (from remote urls)
	for _, list := range conf.Blocklists {
		// get list path
		path := conf.PathToList(list)

		// skip non-remote lists
		if !list.IsRemote() {
			continue
		}

		// skip downloading, don't need to download unless
		// certain conditions are met, which should be triggered
		// from inside the app or similar and not every time
		// an engine is created
		if _, err := os.Stat(path); err == nil {
			continue
		}

		// load/download list
		err := downloader.Download(conf, list)
		if err != nil {
			return nil, err
		}
	}

	// empty activeGroups list of size equal to available groups
	workingGroups := append([]*config.GudgeonGroup{}, conf.Groups...)


	// look for default group
	foundDefaultGroup := false
	for _, group := range conf.Groups {
		if "default" == group.Name {
			foundDefaultGroup = true
			break
		}
	}

	// inject default group
	if !foundDefaultGroup {
		defaultGroup := new(config.GudgeonGroup)
		defaultGroup.Name = "default"
		defaultGroup.Tags = []string{"default"}
		workingGroups = append(workingGroups, defaultGroup)
	}

	// use length of working groups to make list of active groups
	activeGroups := make([]*activeGroup, len(workingGroups))

	// process groups
	for index, group := range conf.Groups {
		// create active group
		activeGroup := new(activeGroup)
		activeGroup.configGroup = group

		// walk through lists and assign to group as needed
		activeGroup.whitelists = shouldAssignList(group.Whitelists, group.Tags, conf.Whitelists)
		activeGroup.blacklists = shouldAssignList(group.Blacklists, group.Tags, conf.Blacklists)
		activeGroup.blocklists = shouldAssignList(group.Blocklists, group.Tags, conf.Blocklists)

		// populate bloom filters as needed
		activeGroup.whiteFilter = createBloomFilter(conf, activeGroup.whitelists)
		activeGroup.blackFilter = createBloomFilter(conf, activeGroup.blacklists)
		activeGroup.blockFilter = createBloomFilter(conf, activeGroup.blocklists)

		// set active group to list of active groups
		activeGroups[index] = activeGroup

		// set default group on engine if found
		if "default" == group.Name {
			engine.defaultGroup = activeGroup
		}
	}

	// attach groups to consumers
	activeConsumers := make([]*activeConsumer, len(conf.Consumers))
	for index, consumer := range conf.Consumers {
		// create an active consumer
		activeConsumer := new(activeConsumer)

		// link consumer to group when the consumer's group elements contains the group name
		for _, activeGroup := range activeGroups {
			if util.StringIn(activeGroup.configGroup.Name, consumer.Groups) {
				activeConsumer.groups = append(activeConsumer.groups, activeGroup)
			}
		}

		// add active consumer to list
		activeConsumers[index] = activeConsumer
	}
	engine.consumers = activeConsumers

	return engine, nil
}

func (engine *engine) consumerGroups(consumer string) []*activeGroup {
	// return the default group in the event nothing else is available
	return []*activeGroup{engine.defaultGroup}
}

func (engine *engine) IsDomainBlocked(consumer string, domain string) bool {

	// get group from conumer string, most likely by converting it to
	// an IP address and then comparing against the items in the consumers list
	// possibly caching the result for later

	// process all group WHITElists

	// process all group BLACKlists

	// query all group bloom filters

	// if in a bloom filter verify against blocklists

	// if nothing blocking the domain is found then 
	// confirm the domain is not blocked
	return false
}

func (engine *engine) Start() error {
	fmt.Printf("Serving %d consumers with a total of %d explicit groups and %d lists", len(engine.consumers), len(engine.config.Groups), len(engine.config.Blacklists) + len(engine.config.Whitelists) + len(engine.config.Blocklists))
	return nil
}