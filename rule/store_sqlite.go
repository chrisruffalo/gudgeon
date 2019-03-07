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
	db, err := sql.Open("sqlite3", sessionDb)
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
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
	db, err := sql.Open("sqlite3", sessionDb+"?mode=ro&_sync=OFF&_mutex=NO&_locking=NORMAL&_journal=MEMORY")
	if err != nil {
		log.Errorf("Rule storage: %s", err)
	}
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

func (store *sqlStore) foundInList(list *config.GudgeonList, domains []string) (bool, string) {
	listName := list.ShortName()

	// convert to interface array for use in query
	params := ""
	dfaces := make([]interface{}, len(domains))
	for idx, domain := range domains {
		if idx > 0 {
			params = params + ", "
		}
		params = fmt.Sprintf("%s$%d", params, idx+1)
		dfaces[idx] = domain
	}

	stmt := "SELECT Rule FROM " + listName + " WHERE Rule in ( " + params + " ) LIMIT 1"
	pstmt, err := store.db.Prepare(stmt)
	defer pstmt.Close()
	if err != nil {
		log.Errorf("Preparing rule storage statement: %s", err)
	}

	rows, err := pstmt.Query(dfaces...)
	defer rows.Close()
	if err != nil {
		log.Errorf("Executing rule storage query: %s", err)
	}

	// check for rows
	if rows == nil || err != nil {
		return false, ""
	}

	// scan rows for rules
	var rule string
	for rows.Next() {
		err = rows.Scan(&rule)
		if "" != rule {
			return true, rule
		}
	}

	return false, ""
}

func (store *sqlStore) FindMatch(lists []*config.GudgeonList, domain string) (Match, *config.GudgeonList, string) {
	// if no block rules initialized we can bail
	if store.db == nil {
		return MatchNone, nil, ""
	}

	// allow and block split
	allowLists := make([]*config.GudgeonList, 0)
	blockLists := make([]*config.GudgeonList, 0)
	for _, l := range lists {
		if ParseType(l.Type) == ALLOW {
			allowLists = append(allowLists, l)
		} else {
			blockLists = append(blockLists, l)
		}
	}

	// get domains
	domains := util.DomainList(domain)

	for _, list := range allowLists {
		if list == nil {
			continue
		}
		if found, rule := store.foundInList(list, domains); found {
			return MatchAllow, list, rule
		}
	}

	for _, list := range blockLists {
		if list == nil {
			continue
		}
		if found, rule := store.foundInList(list, domains); found {
			return MatchBlock, list, rule
		}
	}

	return MatchNone, nil, ""
}

func (store *sqlStore) Close() {
	if store.db != nil {
		store.db.Close()
	}
}