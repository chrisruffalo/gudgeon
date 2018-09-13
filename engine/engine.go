package engine

import (
	"bufio"
	"os"

	"github.com/willf/bloom"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine/downloader"
)

const (
	bloomFilterAcceptableError = float64(0.0005) // rate of acceptable error
)

type activeList struct {
	configList config.GudgeonList
	filter *bloom.BloomFilter

}

type engine struct {
	config *config.GudgeonConfig
	filter *bloom.BloomFilter
	whitelists []*activeList
	blacklists []*activeList
	blocklists []*activeList
}

type Engine interface {
	IsBlocked(domain string) bool
	Start() error
}

func New(config *config.GudgeonConfig) (Engine, error) {
	// make required paths
	os.MkdirAll(config.Paths.Cache, os.ModePerm)

	// create return object
	engine := new(engine)
	engine.config = config

	// load blocklists (from remote urls)
	for _, list := range config.Blocklists {
		// load/download list
		lines, err := downloader.Download(config, list)
		if err != nil {
			return nil, err
		}

		// initialize bloom filter with acceptable error rate
		factor := uint(2)
		filter := bloom.NewWithEstimates(lines * factor, bloomFilterAcceptableError)

		pathTo := config.PathToList(list)
		file, err := os.Open(pathTo)
		if err != nil {
			return nil, err
		}

		// scan file and add to bloom filter
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			filter.AddString(scanner.Text())
		}

		// close file
		file.Close()

		// set bloom filter on active list
		activeList := new(activeList)
		activeList.configList = list
		activeList.filter = filter

		// add active list to engine array
		engine.blocklists = append(engine.blocklists, activeList)
	}

	return engine, nil
}

func (engine *engine) IsBlocked(domain string) bool {

	
	return false
}

func (engine *engine) Start() error {
	return nil
}