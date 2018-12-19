package engine

import (
	"encoding/base64"
	"os"
	"path"

	"github.com/google/uuid"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/downloader"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

// an active group is a group within the engine
// that has been processed and is being used to
// select rules. this will be used with the
// rule processing to create rules and will
// be used by the consumer to talk to the store
type group struct {
	engine *engine

	configGroup *config.GudgeonGroup
}

// represents a parsed "consumer" type that
// links it to active parsed groups
type consumer struct {
	// engine pointer so we can use the engine from the active consumer
	engine *engine

	// configuration that this consumer was parsed from
	configConsumer *config.GundgeonConsumer

	// list of parsed groups that belong to this consumer
	groups     []*group
	groupNames []string
}

// stores the internals of the engine abstraction
type engine struct {
	// the session (which will represent the on-disk location inside of the gudgeon folder)
	// that is being used as backing storage and state behind the engine
	session string

	// maintain config pointer
	config *config.GudgeonConfig

	// consumers that have been parsed
	consumers []*consumer

	// the default group (used to ensure we have one)
	defaultGroup *group

	// the backing store for the engine
	store rule.RuleStore
}

func (engine *engine) Root() string {
	return path.Join(engine.config.SessionRoot(), engine.session)
}

func (engine *engine) ListPath(listType string) string {
	return path.Join(engine.Root(), listType+".list")
}

type Engine interface {
	IsDomainBlocked(consumer string, domain string) bool
	Resolve(consumer string, domain string) string
	Start() error
}

// returns an array of the GudgeonLists that are assigned either by name or by tag from within the list of GudgeonLists in the config file
func assignedLists(listNames []string, listTags []string, lists []*config.GudgeonList) []*config.GudgeonList {
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

	return should
}

func New(conf *config.GudgeonConfig) (Engine, error) {
	// create return object
	engine := new(engine)
	engine.config = conf

	// create store
	engine.store = rule.CreateDefaultStore() // create default store type

	// create session key
	uuid := uuid.New()
	engine.session = base64.RawURLEncoding.EncodeToString([]byte(uuid.String()))

	// make required paths
	os.MkdirAll(conf.Home, os.ModePerm)
	os.MkdirAll(conf.SessionRoot(), os.ModePerm)
	os.MkdirAll(engine.Root(), os.ModePerm)

	// get lists from the configuration
	lists := conf.Lists

	// load lists (from remote urls)
	for _, list := range lists {
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

		// load/download list if required
		err := downloader.Download(conf, list)
		if err != nil {
			return nil, err
		}
	}

	// empty groups list of size equal to available groups
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
	groups := make([]*group, len(workingGroups))

	// process groups
	for _, configGroup := range workingGroups {

		// create active group for gorup name
		engineGroup := new(group)
		engineGroup.engine = engine
		engineGroup.configGroup = configGroup

		// determine which lists belong to this group
		lists := assignedLists(configGroup.Lists, configGroup.Tags, conf.Lists)

		// open the file, read each line, parse to rules
		for _, list := range lists {
			path := conf.PathToList(list)
			array, err := util.GetFileAsArray(path)
			if err != nil {
				continue
			}

			// now parse the array by creating rules and storing them
			parsedType := rule.ParseType(list.Type)
			rules := make([]rule.Rule, len(array))
			for idx, ruleText := range array {
				rules[idx] = rule.CreateRule(ruleText, parsedType)
			}

			// send rule array to engine store
			engine.store.Load(configGroup.Name, rules)
		}

		// set default group on engine if found
		if "default" == configGroup.Name {
			engine.defaultGroup = engineGroup
		}
	}

	// attach groups to consumers
	consumers := make([]*consumer, len(conf.Consumers))
	for index, configConsumer := range conf.Consumers {
		// create an active consumer
		consumer := new(consumer)
		consumer.engine = engine

		// link consumer to group when the consumer's group elements contains the group name
		for _, group := range groups {
			if util.StringIn(group.configGroup.Name, configConsumer.Groups) {
				consumer.groups = append(consumer.groups, group)
				consumer.groupNames = append(consumer.groupNames, group.configGroup.Name)
			}
		}

		// add active consumer to list
		consumers[index] = consumer
	}
	engine.consumers = consumers

	return engine, nil
}

func (engine *engine) consumerGroups(consumer string) []string {
	// return the default group in the event nothing else is available
	return []string{"default"}
}

func (engine *engine) IsDomainBlocked(consumer string, domain string) bool {
	// get groups applicable to consumer
	groupNames := engine.consumerGroups(consumer)
	result := engine.store.IsMatchAny(groupNames, domain)
	return !(result == rule.MatchAllow || result == rule.MatchNone)
}

func (engine *engine) Resolve(consumer string, domain string) string {
	// check cache for answer

	// check block engine to see if it is blocked

	// if blocked, return blocking result (NXDOMAIN, static IP, etc)

	// if not blocked ask upstream DNS

	// todo: don't return empty string
	return ""
}

func (engine *engine) Start() error {
	return nil
}
