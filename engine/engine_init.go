package engine

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/GeertJohan/go.rice"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/db"
	"github.com/chrisruffalo/gudgeon/resolver"
	"github.com/chrisruffalo/gudgeon/rule"
	"github.com/chrisruffalo/gudgeon/util"
)

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

		for _, tag := range list.SafeTags() {
			if util.StringIn(tag, listTags) {
				should = append(should, list)
				break
			}
		}
	}

	return should
}

func createEngineDB(conf *config.GudgeonConfig) (*sql.DB, error) {
	// get path to long-standing data ({home}/'data') and make sure it exists
	dataDir := conf.DataRoot()
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.MkdirAll(dataDir, os.ModePerm)
	}

	// determine path
	dbDir := path.Join(dataDir, "engine")

	// create directory
	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		os.MkdirAll(dbDir, os.ModePerm)
	}

	// get path to db file
	dbPath := path.Join(dbDir, "gudgeon.db")

	// find migrations
	migrationsBox := rice.MustFindBox("migrations")

	// open database
	var err error
	db, err := db.OpenAndMigrate(dbPath, "", migrationsBox)
	if err != nil {
		return nil, fmt.Errorf("Engine DB: %s", err)
	}

	return db, nil
}

func NewEngine(conf *config.GudgeonConfig) (Engine, error) {
	// error collection
	var err error

	// create return object
	engine := new(engine)
	engine.config = conf

	// create session key
	uuid := uuid.New()

	// and make a hidden session folder from  it
	engine.session = "." + base64.RawURLEncoding.EncodeToString([]byte(uuid.String()))

	// make required paths
	err = os.MkdirAll(conf.Home, os.ModePerm)
	if err != nil {
		log.Errorf("Could not create home directory path: %s", err)
	}
	err = os.MkdirAll(conf.SessionRoot(), os.ModePerm)
	if err != nil {
		log.Errorf("Could not create session directory path: %s", err)
	}
	err = os.MkdirAll(engine.Root(), os.ModePerm)
	if err != nil {
		log.Errorf("Could not create engine root directory path: %s", err)
	}

	// configure db if required
	if *conf.Metrics.Enabled || *conf.QueryLog.Enabled {

		//  if persistence functions are enabled, create db
		if (*conf.Metrics.Enabled && *conf.Metrics.Persist) || (*conf.QueryLog.Enabled && *conf.QueryLog.Persist) {
			var err error
			engine.db, err = createEngineDB(conf)
			if err != nil {
				return nil, err
			}
		}

		// build metrics instance (with db if not null)
		if *conf.Metrics.Enabled {
			engine.metrics = NewMetrics(conf, engine.db)
			engine.metrics.UseCacheSizeFunction(engine.CacheSize)
		}

		// build qlog instance (with db if not null)
		if *conf.QueryLog.Enabled {
			engine.qlog, err = NewQueryLog(conf, engine.db)
			if err != nil {
				return nil, err
			}
		}
	}

	// create recorder
	engine.recorder, err = NewRecorder(engine)
	if err != nil {
		return nil, err
	}

	// configure resolvers
	engine.resolvers = resolver.NewResolverMap(conf, conf.Resolvers)

	// get lists from the configuration
	lists := conf.Lists

	// empty groups list of size equal to available groups
	workingGroups := append([]*config.GudgeonGroup{}, conf.Groups...)

	// use length of working groups to make list of active groups
	groups := make([]*group, len(workingGroups))
	groupMap := make(map[string]*group)

	// process groups
	for idx, configGroup := range workingGroups {
		// create active group for gorup name
		engineGroup := new(group)
		engineGroup.engine = engine
		engineGroup.configGroup = configGroup

		// determine which lists belong to this group
		engineGroup.lists = assignedLists(configGroup.Lists, configGroup.SafeTags(), lists)

		// add created engine group to list of groups
		groups[idx] = engineGroup

		// set default group on engine if found
		if "default" == configGroup.Name {
			engine.defaultGroup = engineGroup
		}

		// save group to group map for later reference
		if "" != configGroup.Name {
			groupMap[configGroup.Name] = engineGroup
		}
	}

	// attach groups to consumers
	consumers := make([]*consumer, len(conf.Consumers))
	consumerMap := make(map[string]*consumer)
	for index, configConsumer := range conf.Consumers {
		// create an active consumer
		consumer := &consumer{
			engine:         engine,
			groupNames:     make([]string, 0),
			resolverNames:  make([]string, 0),
			configConsumer: configConsumer,
			lists:          make([]*config.GudgeonList, 0),
		}

		// set as default consumer
		if strings.EqualFold(configConsumer.Name, "default") {
			engine.defaultConsumer = consumer
		}

		// link consumer to group when the consumer's group elements contains the group name
		for _, group := range groups {
			if util.StringIn(group.configGroup.Name, configConsumer.Groups) {
				consumer.groupNames = append(consumer.groupNames, group.configGroup.Name)

				// add resolvers from group too
				if len(group.configGroup.Resolvers) > 0 {
					consumer.resolverNames = append(consumer.resolverNames, group.configGroup.Resolvers...)
				}

				// add lists if they aren't already in the consumer lists
				for _, newList := range group.lists {
					listFound := false
					for _, currentList := range consumer.lists {
						if currentList.CanonicalName() == newList.CanonicalName() {
							listFound = true
							break
						}
					}
					if !listFound {
						consumer.lists = append(consumer.lists, newList)
					}
				}
			}
		}

		// add active consumer to list
		consumers[index] = consumer
		if configConsumer.Name != "" {
			consumerMap[configConsumer.Name] = consumer
		}
	}

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
		err := Download(engine, conf, list)
		if err != nil {
			return nil, err
		}
	}

	// create store based on gudgeon configuration and engine details
	// (requires lists to be downloaded and present before creation)
	totalCount := uint64(0)
	var listCounts []uint64
	engine.store, listCounts = rule.CreateStore(engine.Root(), conf)

	// use/set metrics if they are enabled
	if engine.metrics != nil {
		metrics := engine.metrics
		for idx, list := range conf.Lists {
			log.Infof("List '%s' loaded %d rules", list.CanonicalName(), listCounts[idx])
			rulesCounter := metrics.Get("rules-list-" + list.ShortName())
			rulesCounter.Clear()
			rulesCounter.Inc(int64(listCounts[idx]))
			totalCount += uint64(listCounts[idx])
		}
		totalRulesCounter := metrics.Get(TotalRules)
		totalRulesCounter.Inc(int64(totalCount))
	}

	// set consumers as active on engine
	engine.groups = groupMap
	engine.consumers = consumers
	engine.consumerMap = consumerMap

	// force GC after loading the engine because
	// of all the extra allocation that gets performed
	// during the creation of the arrays and whatnot
	runtime.GC()

	return engine, nil
}
