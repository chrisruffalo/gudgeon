package engine

import (
	"os"
	"github.com/chrisruffalo/gudgeon/config"
)

type engine struct {
	config config.GudgeonConfig
}

type Engine interface {
	Start() error
}

func Prepare(config config.GudgeonConfig) (Engine, error) {
	// make required paths
	os.MkdirAll(config.Paths.Cache, os.ModePerm)

	// download blocklists to cache

	// create bloom filter structure

	// create return object
	engine := new(engine)
	engine.config = config

	return engine, nil
}

func (engine *engine) Start() error {
	return nil
}