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
	// list handling from base store
	baseStore

	// path to storage
	path string

	// db management
	db   *sql.DB
	stmt *sql.Stmt
	tx   *sql.Tx


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

	// clear base store
	store.removeList(list)
}

func (store *sqlStore) Load(list *config.GudgeonList, rule string) {
	var err error

	// add list to base store
	store.addList(list)

	if store.tx == nil {
		store.tx, err = store.db.Begin()
		if err != nil {
			log.Errorf("Could not start transaction: %s", err)
			return
		}
	}

	if store.stmt == nil {
		store.stmt, err = store.tx.Prepare(_insertStmt)
		if err != nil {
			log.Errorf("Could not prepare rule insert statement: %s", err)
			return
		}
	}

	_, err = store.stmt.Exec(list.ShortName(), rule)
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

	if store.stmt != nil {
		_ = store.stmt.Close()
	}

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

func sliceAppend(slice *[]interface{}, value interface{}) {
	*slice = append(*slice, value)
}

func (store *sqlStore) foundInLists(listType config.ListType, lists []*config.GudgeonList, domains []string) (bool, string, string) {
	if store.db == nil {
		return false, "", ""
	}

	// with no lists and no domain we can't test found function
	if len(lists) < 1 || len(domains) < 1 {
		return false, "", ""
	}

	builder := strings.Builder{}

	vars := make([]interface{}, 0, len(lists) + len(domains))
	store.forEachOfTypeIn(listType, lists, func(listType config.ListType, list *config.GudgeonList) {
		sliceAppend(&vars, list.ShortName())
	})

	// no lists selected
	if len(vars) < 1 {
		return false, "", ""
	}

	// build query statement
	builder.WriteString("SELECT l.ShortName, r.Rule FROM rules R LEFT JOIN lists L ON R.ListRowId = L.rowid WHERE l.ShortName in (?")
	builder.WriteString(strings.Repeat(", ?", len(vars)-1))
	builder.WriteString(") AND r.Rule in (?")
	builder.WriteString(strings.Repeat(", ?", len(domains)-1) + ");")

	// build parameters
	for _, dm := range domains {
		vars = append(vars, dm)
	}

	log.Tracef("(type: %d) query: %s (vars: %v)", listType, builder.String(), vars)
	rows, err := store.db.Query(builder.String(), vars...)
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

	// get domains
	domains := util.DomainList(domain)

	if found, listName, rule := store.foundInLists(config.ALLOW, lists, domains); found {
		return MatchAllow, store.getList(listName), rule
	}
	if found, listName, rule := store.foundInLists(config.BLOCK, lists, domains); found {
		return MatchBlock, store.getList(listName), rule
	}

	return MatchNone, nil, ""
}

func (store *sqlStore) Close() {
	if store.db != nil {
		err := store.db.Close()
		if err != nil {
			log.Errorf("Error closing database: %s", err)
		}
		store.db = nil
	}
}
