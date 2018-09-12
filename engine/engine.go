package engine

import (
	"os"
	"path/filepath"
	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/engine/database"
)

type engine struct {
	config config.GudgeonConfig
	db database.GudgeonDatabase
}

type Engine interface {
	Start() error
}

func Prepare(config config.GudgeonConfig) (Engine, error) {
	// make required paths
	os.MkdirAll(config.Paths.Cache, os.ModePerm)
	os.MkdirAll(filepath.Dir(config.Paths.Database), os.ModePerm)

	// init db
	db, err := database.Get(config.Paths.Database)
	if err != nil {
		return nil, err
	}	

	// download blocklists to cache

	// process downloaded blocklists into db

	// create bloom filter structure

	// create return object
	engine := new(engine)
	engine.db = db
	engine.config = config

	return engine, nil
}

func (engine *engine) Start() error {
	return nil
}