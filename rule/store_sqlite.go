package rule

import (
	"database/sql"
	"os"
	"path"
	"strings"

	"github.com/GeertJohan/go.rice"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/db"
	"github.com/chrisruffalo/gudgeon/util"
)

// the static names of the rules database
const sqlDbName = "rules.db"
const _insertStmt = "INSERT OR IGNORE INTO rules (ListRowId, Rule) VALUES ((SELECT Id FROM lists WHERE ShortName = ? LIMIT 1), ?)"
const _deleteStmt = "DELETE FROM rules WHERE ListRowId = (SELECT Id FROM lists WHERE ShortName = ? LIMIT 1)"

type sqlStore struct {
	path string
	db   *sql.DB

	tx *sql.Tx
}

func (store *sqlStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	// get session storage location
	sessionDb := path.Join(sessionRoot, sqlDbName)
	if _, err := os.Stat(sessionRoot); os.IsNotExist(err) {
		err = os.MkdirAll(sessionRoot, os.ModePerm)
		if err != nil {
			log.Errorf("Could not create directories for SQL storage: %s", err)
		}
	}
	store.path = sessionDb

	// get/migrate schema
	migrationsBox := rice.MustFindBox("sqlite-store-migrations")

	// open db with migrated schema
	var err error
	store.db, err = db.OpenAndMigrate(store.path, "cache=shared&_journal_mode=OFF", migrationsBox)
	if err != nil {
		log.Errorf("Creating SQLite Rule Store: %s", err)
	}
	// because this db doesn't read and write at the same time (or at any time)
	// this is safe and doesn't have much of an impact if you set it higher
	// anyway
	store.db.SetMaxOpenConns(1)

	// insert lists into table
	for _, list := range lists {
		if list == nil {
			continue
		}
		_, err = store.db.Exec("INSERT INTO lists (ShortName) VALUES (?)", list.ShortName())
		if err != nil {
			log.Errorf("Inserting list: %s", err)
		}
	}
}

func (store *sqlStore) Clear(config *config.GudgeonConfig, list *config.GudgeonList) {
	// close transaction if it exists
	if store.tx != nil {
		err := store.tx.Commit()
		if err != nil {
			log.Errorf("Committing rules to rules DB: %s", err)
			err = store.tx.Rollback()
			if err != nil {
				log.Errorf("Rolling back rule clear transaction: %s", err)
			}
		}
		log.Tracef("Closing initial transaction...")
	}

	var err error

	store.tx, err = store.db.Begin()
	if err != nil {
		log.Errorf("Could not start transaction: %s", err)
		return
	}

	_, err = store.tx.Exec(_deleteStmt, list.ShortName())
	if err != nil {
		log.Errorf("Could not insert into rules store: %s", err)
		err = store.tx.Rollback()
		if err != nil {
			log.Errorf("Could not roll back transaction: %s", err)
		}
	}

	err = store.tx.Commit()
	if err != nil {
		log.Errorf("Could not close rule clear statement: %s", err)
	}
	// start with a fresh transaction on the next operation
	store.tx = nil
}

func (store *sqlStore) Load(list *config.GudgeonList, rule string) {
	if store.tx == nil {
		var err error
		store.tx, err = store.db.Begin()
		if err != nil {
			log.Errorf("Could not start transaction: %s", err)
			return
		}
	}

	_, err := store.tx.Exec(_insertStmt, list.ShortName(), rule)
	if err != nil {
		log.Errorf("Could not insert into rules store: %s", err)
		err = store.tx.Rollback()
		if err != nil {
			log.Errorf("Could not roll back transaction: %s", err)
		}
		store.tx = nil
	}
}

func (store *sqlStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	var err error

	// commit any outstanding transactions
	if store.tx != nil {
		err = store.tx.Commit()
		if err != nil {
			log.Errorf("Committing index transaction: %s", err)
			err = store.tx.Rollback()
			if err != nil {
				log.Errorf("Rolling back transaction: %s", err)
			}
		}
		// clear out for further transactions
		store.tx = nil
	}

	// free memory
	_, err = store.db.Exec("PRAGMA shrink_memory;")
	if err != nil {
		log.Errorf("Could not shrink memory in db finalize")
	}
}

func (store *sqlStore) foundInLists(lists []*config.GudgeonList, domains []string) (bool, string, string) {
	// with no lists and no domain we can't test found function
	if len(lists) < 1 || len(domains) < 1 {
		return false, "", ""
	}

	shortNames := make([]string, 0, len(lists))
	for _, list := range lists {
		shortNames = append(shortNames, list.ShortName())
	}

	// build query statement
	stmt := "SELECT l.ShortName, r.Rule FROM rules R LEFT JOIN lists L ON R.ListRowId = L.rowid WHERE l.ShortName in (?" + strings.Repeat(", ?", len(shortNames)-1) + ") AND r.Rule in (?" + strings.Repeat(", ?", len(domains)-1) + ");"

	// build parameters
	vars := make([]interface{}, 0, len(shortNames)+len(domains))
	for _, sn := range shortNames {
		vars = append(vars, sn)
	}
	for _, dm := range domains {
		vars = append(vars, dm)
	}

	rows, err := store.db.Query(stmt, vars...)
	if err != nil {
		log.Errorf("Executing rule storage query: %s", err)
	}
	defer rows.Close()

	// check for rows
	if rows == nil || err != nil {
		return false, "", ""
	}

	// scan rows for rules
	var list string
	var rule string
	for rows.Next() {
		err = rows.Scan(&list, &rule)
		if err != nil {
			log.Errorf("Rule row scan: %s", err)
			continue
		}
		if "" != rule {
			return true, list, rule
		}
	}

	return false, "", ""
}

func (store *sqlStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {
	// if no block rules initialized we can bail
	if store.db == nil {
		return MatchNone, nil, ""
	}

	// allow and block split
	listmap := make(map[string]*config.GudgeonList)
	allowLists := make([]*config.GudgeonList, 0)
	blockLists := make([]*config.GudgeonList, 0)
	for _, l := range lists {
		if l == nil {
			continue
		}
		if ParseType(l.Type) == ALLOW {
			allowLists = append(allowLists, l)
		} else {
			blockLists = append(blockLists, l)
		}
		listmap[l.ShortName()] = l
	}

	// get domains
	domains := util.DomainList(domain)

	if found, listName, rule := store.foundInLists(allowLists, domains); found {
		return MatchAllow, listmap[listName], rule
	}
	if found, listName, rule := store.foundInLists(blockLists, domains); found {
		return MatchBlock, listmap[listName], rule
	}

	return MatchNone, nil, ""
}

func (store *sqlStore) Close() {
	if store.db != nil {
		err := store.db.Close()
		if err != nil {
			log.Errorf("Error closing database: %s", err)
		}
	}
}
