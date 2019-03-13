package rule

import (
	"database/sql"
	"fmt"
	"os"
	"path"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	log "github.com/sirupsen/logrus"

	"github.com/chrisruffalo/gudgeon/config"
	"github.com/chrisruffalo/gudgeon/util"
)

// practically sqlite only supports a parameter list up to 999 characters so we set this here
// going over that requires abandoning the prepared statements and using string building and
// direct insertion
const sqlBatchSize = 999
const sqlDbName = "rules.db"

type sqlStore struct {
	db      *sql.DB
	batches map[string][]string
	ptr     map[string]int
}

func (store *sqlStore) Init(sessionRoot string, config *config.GudgeonConfig, lists []*config.GudgeonList) {
	// create batches map and pointer map
	store.batches = make(map[string][]string)
	store.ptr = make(map[string]int)

	// get session storage location
	sessionDb := path.Join(sessionRoot, sqlDbName)
	if _, err := os.Stat(sessionRoot); os.IsNotExist(err) {
		os.MkdirAll(sessionRoot, os.ModePerm)
	}
	db, err := sql.Open("sqlite3", sessionDb + "?cache=shared&journal_mode=WAL")
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
	db.SetMaxOpenConns(1)
	store.db = db

	// pre-create tabels
	for _, list := range lists {
		// init empty lists
		store.batches[list.ShortName()] = make([]string, sqlBatchSize)

		// init pointers
		store.ptr[list.ShortName()] = -1

		stmt := "CREATE TABLE IF NOT EXISTS " + list.ShortName() + " ( Rule TEXT );"
		_, err := store.db.Exec(stmt)
		if err != nil {
			log.Errorf("Rule storage: %s", err)
		}
	}
}

func (store *sqlStore) insert(listName string, rules []string) {
	if len(rules) < 1 {
		return
	}
	stmt := "INSERT INTO " + listName + " (RULE) VALUES (?)" + strings.Repeat(", (?)", len(rules)-1)
	vars := make([]interface{}, len(rules))
	for idx, _ := range rules {
		vars[idx] = rules[idx]
	}

	pstmt, err := store.db.Prepare(stmt)
	if err != nil {
		log.Errorf("Preparing rule storage statement: %s", err)
		return
	}
	defer pstmt.Close()

	_, err = pstmt.Exec(vars...)
	if err != nil {
		log.Errorf("During rule storage insert: %s", err)
	}
}

func (store *sqlStore) Load(list *config.GudgeonList, rule string) {
	listName := list.ShortName()

	store.batches[listName][store.ptr[listName]+1] = rule
	store.ptr[listName] = store.ptr[listName] + 1

	// if enough items are in the batch, insert
	if store.ptr[listName] >= sqlBatchSize-1 {
		store.insert(listName, store.batches[listName])
		store.ptr[listName] = -1
	}
}

func (store *sqlStore) Finalize(sessionRoot string, lists []*config.GudgeonList) {
	// close and re-open db
	store.db.Close()
	sessionDb := path.Join(sessionRoot, sqlDbName)
	db, err := sql.Open("sqlite3", sessionDb+"?mode=ro&cache=shared")
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
	db.SetMaxOpenConns(1)
	store.db = db

	for _, list := range lists {
		listName := list.ShortName()
		// finalize inserts
		if store.ptr[listName] >= 0 {
			store.insert(listName, store.batches[listName][0:store.ptr[listName]+1])
			delete(store.batches, listName)
			delete(store.ptr, listName)
		}

		// after everything is inserted, add indexes in one go
		idxStmt := "CREATE INDEX IF NOT EXISTS idx_" + listName + "_Rule ON " + listName + " (Rule);"
		_, err := store.db.Exec(idxStmt)
		if err != nil {
			log.Errorf("Could not create index on table %s: %s", listName, err)
		}
	}
}

func (store *sqlStore) foundInLists(lists []*config.GudgeonList, domains []string) (bool, string, string) {
	// with no lists and no domain we can't test found function
	if len(lists) < 1 || len(domains) < 1 {
		return false, "", ""
	}

	// convert to interface array for use in query
	params := ""
	dfaces := make([]interface{}, len(domains))
	for idx, domain := range domains {
		if idx > 0 {
			params = params + ", "
		}
		params = params + "?"
		dfaces[idx] = domain
	}

	subselects := make([]string, 0, len(lists))
	for _, list := range lists {
		subselects = append(subselects, fmt.Sprintf("SELECT '%s' as List, Rule FROM %s", list.ShortName(), list.ShortName()))
	}

	var stmt string
	if len(subselects) == 1 {
		stmt = subselects[0] + " WHERE Rule in ( " + params + " ) LIMIT 1"
	} else {
		stmt = "SELECT List, Rule FROM (" + strings.Join(subselects, " UNION ") + ") WHERE Rule in ( " + params + " ) LIMIT 1"
	}

	pstmt, err := store.db.Prepare(stmt)
	defer pstmt.Close()
	if err != nil {
		log.Errorf("Preparing rule query statement: %s", err)
	}

	rows, err := pstmt.Query(dfaces...)
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
