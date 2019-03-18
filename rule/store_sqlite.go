package rule

import (
	"database/sql"
	//"fmt"
	"os"
	"path"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

// how many rules to take in before committing the transaction
const txBatchSize = 75000
// the static names of the rules database
const sqlDbName = "rules.db"

type sqlStore struct {
	stmt    *sql.Stmt
	db      *sql.DB
	tx      *sql.Tx
	batch   int
}

func (store *sqlStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	// get session storage location
	sessionDb := path.Join(sessionRoot, sqlDbName)
	if _, err := os.Stat(sessionRoot); os.IsNotExist(err) {
		os.MkdirAll(sessionRoot, os.ModePerm)
	}
	db, err := sql.Open("sqlite3", sessionDb+"?cache=shared&journal_mode=WAL")
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
	db.SetMaxOpenConns(1)
	store.db = db

	_, err = store.db.Exec("CREATE TABLE IF NOT EXISTS lists ( Id INTEGER PRIMARY KEY, ShortName TEXT )")
	if err != nil {
		log.Errorf("Creating list table: %s", err)
	}
	_, err = store.db.Exec("CREATE INDEX IF NOT EXISTS idx_lists_ShortName on lists (ShortName)")
	if err != nil {
		log.Errorf("Creating list name index: %s", err)
	}	

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

	_, err = store.db.Exec("CREATE TABLE IF NOT EXISTS rules ( ListRowId INTEGER, Rule TEXT, PRIMARY KEY(ListRowId, Rule) ) WITHOUT ROWID")
	if err != nil {
		log.Errorf("Creating rule storage: %s", err)
	}

	_, err = store.db.Exec("CREATE INDEX IF NOT EXISTS idx_rules_Rule on rules (Rule)")
	if err != nil {
		log.Errorf("Creating rule index: %s", err)
	}
	_, err = store.db.Exec("CREATE INDEX IF NOT EXISTS idx_rules_ListRowId on rules (ListRowId)")
	if err != nil {
		log.Errorf("Creating rule index: %s", err)
	}
	_, err = store.db.Exec("CREATE INDEX IF NOT EXISTS idx_rules_IdRule on rules (ListRowId, Rule)")
	if err != nil {
		log.Errorf("Creating rule index: %s", err)
	}		
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

	if store.tx != nil && store.batch > txBatchSize {
		err := store.tx.Commit()
		if err != nil {
			log.Errorf("Commiting rules to rules DB: %s", err)
			return
		}

		// close statment
		if store.stmt != nil {
			store.stmt.Close()
			store.stmt = nil
		}

		// restart batch
		store.batch = 0

		// restart transaction
		store.tx, err = store.db.Begin()
		if err != nil {
			log.Errorf("Could not start transaction: %s", err)
			return
		}		
	}

	// only prepare the statement when it is nil (after batch insert)
	if store.stmt == nil {
		var err error

		stmt := "INSERT OR IGNORE INTO rules (ListRowId, Rule) VALUES ((SELECT Id FROM lists WHERE ShortName = ? LIMIT 1), ?)"
		store.stmt, err = store.tx.Prepare(stmt)
		if err != nil {
			log.Errorf("Preparing rule storage statement: %s", err)
			return
		}
	}

	_, err := store.stmt.Exec(list.ShortName(), rule)
	if err != nil {
		store.tx.Rollback()
		log.Errorf("Could not insert into rules store: %s", err)
		store.tx = nil
		store.stmt = nil
	}

	// increase batch size
	store.batch++
}

func (store *sqlStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	// clean up statement
	if store.stmt != nil {
		store.stmt.Close()
		store.stmt = nil
	}

	// close transaction if it exists
	if store.tx != nil {
		err := store.tx.Commit()
		if err != nil {
			log.Errorf("Commiting rules to rules DB: %s", err)
		}
		// clean up after
		store.tx = nil 
	}

	// close and re-open db
	store.db.Close()
	sessionDb := path.Join(sessionRoot, sqlDbName)
	db, err := sql.Open("sqlite3", sessionDb+"?mode=ro&cache=shared")
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
	db.SetMaxOpenConns(1)
	store.db = db
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
	stmt := "SELECT l.ShortName, r.Rule FROM rules R LEFT JOIN lists L ON R.ListRowId = L.rowid WHERE l.ShortName in (?" + strings.Repeat(", ?", len(shortNames) - 1 ) + ") AND r.Rule in (?" + strings.Repeat(", ?", len(domains) - 1) + ");"
	pstmt, err := store.db.Prepare(stmt)
	defer pstmt.Close()
	if err != nil {
		log.Errorf("Preparing rule query statement: %s", err)
	}

	// build parameters
	vars := make([]interface{}, 0, len(shortNames) + len(domains))
	for _, sn := range shortNames {
		vars = append(vars, sn)
	}
	for _, dm := range domains {
		vars = append(vars, dm)	
	}	

	rows, err := pstmt.Query(vars...)
	defer rows.Close()
	if err != nil {
		log.Errorf("Executing rule storage query: %s", err)
	}

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
		store.db.Close()
	}
}
