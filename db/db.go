package db

import (
	"database/sql"
	"fmt"
	"github.com/GeertJohan/go.rice"
	"github.com/atrox/go-migrate-rice"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	DefaultOptions = "cache=shared&_journal_mode=WAL"
	_dbDriver      = "sqlite3"
	_dbLifetime    = time.Minute
	_dbMaxConn     = 2
	_dbMaxIdle     = 0
)

func Open(path string, options string) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("Opening database '%s': path must not be empty", path)
	}

	if options == "" {
		options = DefaultOptions
	}

	db, err := sql.Open(_dbDriver, path+"?"+options)
	if err != nil {
		return nil, err
	}

	// set connection info from constants
	db.SetConnMaxLifetime(_dbLifetime)
	db.SetMaxOpenConns(_dbMaxConn)
	db.SetMaxIdleConns(_dbMaxIdle)

	return db, nil
}

func OpenAndMigrate(path string, options string, box *rice.Box) (*sql.DB, error) {
	db, err := Open(path, options)
	if err != nil {
		return nil, err
	}

	migrationDriver, err := migraterice.WithInstance(box)
	if err != nil {
		log.Errorf("Could not get migration instances: %s", err)
		return nil, err
	}

	dbDriver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Errorf("Could not open db: %s", err)
		return nil, err
	}

	m, err := migrate.NewWithInstance("rice", migrationDriver, "sqlite3", dbDriver)
	if err != nil {
		log.Errorf("Could not migrate: %s", err)
		return nil, err
	}

	// migrate to newest version of database in the given box
	err = m.Up()
	// only return error if it is not a "no change" error (those are fine)
	if err != nil && err != migrate.ErrNoChange {
		return nil, err
	}

	// free memory
	_, err = db.Exec(`PRAGMA shrink_memory;`)
	if err != nil {
		log.Errorf("Could not shrink memory after db migrate: %s", err)
	}

	// return the migrated database
	return db, nil
}
