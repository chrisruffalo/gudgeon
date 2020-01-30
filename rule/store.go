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

	// decent size buffer
	_loadBufferSize = 32 * 1024
)

type Store interface {
	Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList)

	Load(list *config.GudgeonList, rule string)

	Clear(config *config.GudgeonConfig, list *config.GudgeonList)

	Finalize(sessionRoot string, lists []*config.GudgeonList)

	FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string)

	Close()
}

type baseStore struct {
	// map of block/allow -> short name -> config list
	lists map[config.ListType]map[string]*config.GudgeonList
}

type eachListAction func(index int, listType config.ListType, list *config.GudgeonList)
type matchListAction func(listType config.ListType, list *config.GudgeonList) (Match, *config.GudgeonList, string)

func (baseStore *baseStore) addList(list *config.GudgeonList) {
	if list == nil {
		return
	}
	if baseStore.lists == nil {
		baseStore.lists = make(map[config.ListType]map[string]*config.GudgeonList)
	}
	if _, found := baseStore.lists[list.ParsedType()]; !found {
		baseStore.lists[list.ParsedType()] = make(map[string]*config.GudgeonList)
	}
	baseStore.lists[list.ParsedType()][list.ShortName()] = list
}

func (baseStore *baseStore) removeList(list *config.GudgeonList) {
	if list == nil {
		return
	}
	if baseStore.lists == nil {
		return
	}
	if _, found := baseStore.lists[list.ParsedType()]; !found {
		return
	}
	delete(baseStore.lists[list.ParsedType()], list.ShortName())
}

func (baseStore *baseStore) getList(name string) *config.GudgeonList {
	if "" == name {
		return nil
	}
	for _, v := range baseStore.lists {
		if _, ok := v[name]; ok {
			return v[name]
		}
	}
	return nil
}

/**
 * For each list of the given type in the base store that also appears in the list that was given
 * perform an action.
 */
func (baseStore *baseStore) forEachOfTypeIn(listType config.ListType, lists []*config.GudgeonList, listAction eachListAction) int {
	// lists that have no elements or nil functions can't match anything
	if len(lists) < 1 || listAction == nil {
		return 0
	}
	// if no lists of the given type exist in the base store then no matches will be found
	if _, found := baseStore.lists[listType]; !found {
		return 0
	}
	actionedListCount := 0
	for _, v := range lists {
		if list, found := baseStore.lists[listType][v.ShortName()]; found {
			listAction(actionedListCount, listType, list)
			actionedListCount++
		}
	}
	return actionedListCount
}

func (baseStore *baseStore) forEachIn(lists []*config.GudgeonList, listAction eachListAction) int {
	// lists that have no elements or nil functions can't match anything
	if len(lists) < 1 || listAction == nil {
		return 0
	}
	actionedListCount := 0
	var found *config.GudgeonList
	for _, list := range lists {
		found = baseStore.getList(list.ShortName())
		if found != nil {
			listAction(actionedListCount, found.ParsedType(), found)
			actionedListCount++
		}
	}
	return actionedListCount
}

/**
 * For each list of the given type in the base store that also appears in the list that was given
 * perform an action that can return a match
 */
func (baseStore *baseStore) matchForEachOfTypeIn(listType config.ListType, lists []*config.GudgeonList, matchAction matchListAction) (Match, *config.GudgeonList, string) {
	// if there are no actions no match can happen anyway
	if len(lists) < 1 || matchAction == nil {
		return MatchNone, nil, ""
	}
	// if no lists of the given type exist in the base then no matches can be found for any of the lists
	if _, found := baseStore.lists[listType]; !found {
		return MatchNone, nil, ""
	}
	for _, v := range lists {
		if list, found := baseStore.lists[listType][v.ShortName()]; found {
			m, list, rule := matchAction(listType, list)
			if MatchNone != m {
				return m, list, rule
			}
		}
	}
	return MatchNone, nil, ""
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
	if "bloomhash" == backingStoreType {
		bloomStore := new(bloomStore)
		bloomStore.backingStore = new(hashStore)
		delegate = bloomStore
	} else if "hash" == backingStoreType || "hash64" == backingStoreType {
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

	// buffer for reading files
	var buffer = make([]byte, _loadBufferSize)

	for _, list := range conf.Lists {
		listCounter := loadList(store, conf, list, buffer)

		// locally scoped variable for list watching
		watchList := list

		// notify that we want to watch for changes in a given file
		events.Send("file:watch:start", &events.Message{"path": conf.PathToList(watchList)})
		// save handle so it can later be used to close watchers
		handle := events.Listen("file:"+conf.PathToList(watchList), func(message *events.Message) {
			store.Clear(conf, watchList)
			newRuleCount := loadList(store, conf, watchList, buffer)
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

// load list with a reusable buffer
func loadList(store Store, config *config.GudgeonConfig, list *config.GudgeonList, buffer []byte) uint64 {
	// open file and scan
	data, err := os.Open(config.PathToList(list))
	if err != nil {
		log.Errorf("Could not open list file: %s", err)
		return uint64(0)
	}

	listCounter := uint64(0)

	// scan through file
	scanner := bufio.NewScanner(data)
	scanner.Buffer(buffer, len(buffer))
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
