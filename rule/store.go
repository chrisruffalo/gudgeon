package rule

import (
	"bufio"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/events"
)

// a match can be:
// allow (don't block, override/bypass block)
// block (explicit block)
// none (no reason found to block or allow)
type Match uint8

const (
	MatchAllow Match = 2
	MatchBlock Match = 1
	MatchNone  Match = 0
)

type Store interface {
	Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList)

	Load(list *config.GudgeonList, rule string)

	Clear(config *config.GudgeonConfig, list *config.GudgeonList)

	Finalize(sessionRoot string, lists []*config.GudgeonList)

	FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string)

	Close()
}

// stores are created from lists of files inside a configuration
func CreateStore(storeRoot string, conf *config.GudgeonConfig) (Store, []uint64) {
	// outer shell reloading store
	store := &reloadingStore{
		handlers: make([]*events.Handle, 0),
	}

	// get type of backing store from conf file
	backingStoreType := strings.ToLower(conf.Storage.RuleStorage)

	// create appropriate backing store
	var delegate Store
	if "hash" == backingStoreType || "hash64" == backingStoreType {
		delegate = new(hashStore)
		backingStoreType = "hash"
	} else if "hash+sqlite" == backingStoreType {
		hashStore := new(hashStore)
		hashStore.delegate = new(sqlStore)
		delegate = hashStore
	} else if "hash32" == backingStoreType {
		delegate = new(hashStore32)
	} else if "hash32+sqlite" == backingStoreType {
		hashStore32 := new(hashStore32)
		hashStore32.delegate = new(sqlStore)
		delegate = hashStore32
	} else if "sqlite" == backingStoreType || "sql" == backingStoreType {
		delegate = new(sqlStore)
		backingStoreType = "sqlite"
	} else if "bloom" == backingStoreType {
		delegate = new(bloomStore)
	} else if "bloom+sqlite" == backingStoreType || "bloom+sql" == backingStoreType {
		bloomStore := new(bloomStore)
		bloomStore.backingStore = new(sqlStore)
		delegate = bloomStore
	} else {
		if backingStoreType != "memory" && backingStoreType != "mem" && backingStoreType != "" {
			log.Warnf("Could not find backing store type '%s', using default memory store instead", backingStoreType)
		}
		delegate = new(memoryStore)
		backingStoreType = "memory"
	}
	log.Infof("Using '%s' rule store implementation", backingStoreType)

	// for our outer reloading delegate to a complex store that delegates to the type of chosen store
	// reloading -> complex -> actual chosen store (which can delegate even further)
	store.delegate = &complexStore{backingStore: delegate}

	// initialize stores
	store.Init(storeRoot, conf, conf.Lists)

	// load files into stores based on complexity
	outputCount := make([]uint64, 0, len(conf.Lists))

	for _, list := range conf.Lists {
		listCounter := loadList(store, conf, list)

		// locally scoped variable for list watching
		watchList := list

		// notify that we want to watch for changes in a given file
		events.Send("file:watch:start", &events.Message{"path": conf.PathToList(watchList)})
		// save handle so it can later be used to close watchers
		handle := events.Listen("file:"+conf.PathToList(watchList), func(message *events.Message) {
			store.Clear(conf, watchList)
			newRuleCount := loadList(store, conf, watchList)
			store.Finalize(conf.SessionRoot(), []*config.GudgeonList{watchList})
			// send message that a list value changed
			events.Send("store:list:changed", &events.Message{
				"listName":      watchList.CanonicalName(),
				"listShortName": watchList.ShortName(),
				"count":         newRuleCount,
			})
			// watch file again
			events.Send("file:watch:start", &events.Message{"path": conf.PathToList(watchList)})
		})
		if handle != nil {
			store.handlers = append(store.handlers, handle)
		}

		// append counter to output count
		outputCount = append(outputCount, listCounter)
	}

	// finalize both stores (store finalizes delegate)
	store.Finalize(storeRoot, conf.Lists)

	// finalize and return store
	return store, outputCount
}

func loadList(store Store, config *config.GudgeonConfig, list *config.GudgeonList) uint64 {
	// open file and scan
	data, err := os.Open(config.PathToList(list))
	if err != nil {
		log.Errorf("Could not open list file: %s", err)
		return uint64(0)
	}

	listCounter := uint64(0)

	// scan through file
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		text := ParseLine(scanner.Text())
		if "" != text {
			// load the text into the store which will load it into the next delegate
			// if it doesn't match the parameters of that store
			store.Load(list, text)
			listCounter++
		}
	}

	// close file
	err = data.Close()
	if err != nil {
		log.Errorf("Could not close file: %s", err)
	}

	return listCounter
}
