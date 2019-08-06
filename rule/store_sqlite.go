package rule

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/db"
	"github.com/chrisruffalo/gudgeon/util"
)

// the static names of the rules database
const defaultVarSize = 10
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

	// insert var pooling
	varPool sync.Pool

	// statement caching
	// map of number of lists -> number of domains -> query with appropriate number of variable slots
	// seems like overkill but easily paid off by the sheer number of queries that are performed and
	// how often the cache should "hit"
	stmtCache *cache.Cache
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

	// create var pool
	store.varPool = sync.Pool{
		New: func() interface{} {
			return make([]interface{}, 0, defaultVarSize)
		},
	}

	// set up cache
	store.stmtCache = cache.New(time.Minute*5, time.Minute)
	// and evict items on close
	store.stmtCache.OnEvicted(func(s string, i interface{}) {
		if stmt, ok := i.(*sql.Stmt); ok {
			err := stmt.Close()
			if err != nil {
				log.Errorf("During close/evict: %s", err)
			}
		}
	})
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

const (
	queryStartFragment = "SELECT l.ShortName, r.Rule FROM rules R LEFT JOIN lists L ON R.ListRowId = L.rowid WHERE l.ShortName in (?"
	queryMidFragment   = ") AND r.Rule in (?"
)

func (store *sqlStore) foundInLists(listType config.ListType, lists []*config.GudgeonList, domains []string) (bool, string, string) {
	if store.db == nil {
		return false, "", ""
	}

	// with no lists and no domain we can't test found function
	if len(lists) < 1 || len(domains) < 1 {
		return false, "", ""
	}

	numDomains := len(domains)

	vars := store.varPool.Get().([]interface{})
	defer store.varPool.Put(vars)
	numLists := store.forEachOfTypeIn(listType, lists, func(listType config.ListType, list *config.GudgeonList) {
		sliceAppend(&vars, list.ShortName())
	})

	// no lists selected
	if numLists < 1 {
		return false, "", ""
	}

	var stmt *sql.Stmt
	key := fmt.Sprintf("%d|%d", numLists, numDomains)
	if i, found := store.stmtCache.Get(key); found {
		if v, ok := i.(*sql.Stmt); ok {
			// get itme
			stmt = v
			// overwrite duration (so timer restarts)
			store.stmtCache.SetDefault(key, stmt)
		}
	}

	if nil == stmt {
		// build query statement
		builder := strings.Builder{}
		builder.WriteString(queryStartFragment)
		builder.WriteString(strings.Repeat(", ?", numLists-1))
		builder.WriteString(queryMidFragment)
		builder.WriteString(strings.Repeat(", ?", numDomains-1) + ");")

		// prepare statement
		var err error
		// save to cache
		stmt, err = store.db.Prepare(builder.String())
		if err != nil {
			log.Errorf("Preparing cached query statement: %s", err)
			return false, "", ""
		}

		// set with default expiration
		store.stmtCache.SetDefault(key, stmt)
	}

	// build parameters
	for _, dm := range domains {
		vars = append(vars, dm)
	}

	rows, err := stmt.Query(vars[:numLists+numDomains]...)
	if err != nil {
		log.Errorf("Executing rule check query: %s", err)
	}
	// check for rows
	if rows == nil || err != nil {
		return false, "", ""
	}
	defer rows.Close()

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
		// close prepared statements
		for _, i := range store.stmtCache.Items() {
			if stmt, ok := i.Object.(*sql.Stmt); ok {
				err := stmt.Close()
				if err != nil {
					log.Errorf("Closing sql rule query database/statement: %s", err)
				}
			}
		}
		store.stmtCache.Flush()

		err := store.db.Close()
		if err != nil {
			log.Errorf("Error closing database: %s", err)
		}
		store.db = nil
	}
}
