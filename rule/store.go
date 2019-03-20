package rule

import (
	"bufio"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
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

type RuleStore interface {
	Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList)

	Load(list *config.GudgeonList, rule string)

	Finalize(sessionRoot string, lists []*config.GudgeonList)

	FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string)

	Close()
}

// stores are created from lists of files inside a configuration
func CreateStore(storeRoot string, config *config.GudgeonConfig) (RuleStore, []uint64) {
	// first create the complex rule store wrapper
	store := new(complexStore)

	// get type of backing store from config file
	backingStoreType := strings.ToLower(config.Storage.RuleStorage)

	// create appropriate backing store
	var delegate RuleStore
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

	// set backing store
	store.backingStore = delegate

	// initialize stores
	store.Init(storeRoot, config, config.Lists)

	// load files into stores based on complexity
	outputCount := make([]uint64, 0, len(config.Lists))

	for _, list := range config.Lists {
		// open file and scan
		data, err := os.Open(config.PathToList(list))
		if err != nil {
			data.Close()
			log.Errorf("Could not open list file: %s", err)
			outputCount = append(outputCount, 0)
			continue
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
				listCounter += 1
			}
		}

		// close file
		data.Close()

		// append counter to output count
		outputCount = append(outputCount, listCounter)
	}

	// finalize both stores (store finalizes delegate)
	store.Finalize(storeRoot, config.Lists)

	// finalize and return store
	return store, outputCount
}