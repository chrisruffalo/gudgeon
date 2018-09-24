package benchmarks

import (
  	"database/sql"
 	"io/ioutil"
 	"os"
 	"strings"

    _ "github.com/mattn/go-sqlite3"
)

type sqlstore struct {
	file string
	db *sql.DB

	insertCacheStmtSize int
	insertCacheStmt *sql.Stmt

	stmt *sql.Stmt
}

const (
	//insertValueSize * batch_size should never be greater than OR EQUSL TO 1000
	batch_size = 498

	insertStmt = "INSERT INTO rules (rule, groups) VALUES "
	insertValues = "(?, ?)"
	insertValueSize = 2

	searchQuery = "SELECT rule FROM rules WHERE rule in (?, ?) LIMIT 1;"
)

func (sqlstore *sqlstore) Id() string {
	return "sql: '" + sqlstore.file + "'"
}

func (sqlstore *sqlstore) insert(db *sql.DB, tx *sql.Tx, rules []string) error {
	// build value-statement if the size of the value statement has changed (or non-existant)
	if sqlstore.insertCacheStmt == nil || sqlstore.insertCacheStmtSize != len(rules) {
		s := insertStmt + insertValues + strings.Repeat(", " + insertValues, len(rules) - 1)

		stmt, err := db.Prepare(s)
		if err != nil {
			return err
		}

		// cache statement and statement size
		sqlstore.insertCacheStmt = stmt
		sqlstore.insertCacheStmtSize = len(rules)
	}
	
	args := make([]interface{}, len(rules) * insertValueSize)
	ridx := 0
	for idx := 0; idx < len(args); idx += insertValueSize {
		args[idx] = rules[ridx]
		args[idx+1] = "alpha, bravo, charlie, delta"
		ridx++
	}

	_, err := tx.Stmt(sqlstore.insertCacheStmt).Exec(args...)
	if err != nil {
		return err
	}

	return nil
}

func (sqlstore *sqlstore) Load(inputfile stringd) error {
	// go through file
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}

	// use this method to comb the array for bad items before it 
	// is used in the batch insert
	array := strings.Split(string(content), "\r")
	mindex := 0
	for _, item := range array {
		item = strings.TrimSpace(item)
		if "" == item {
			continue
		}
		array[mindex] = item
		mindex++
	}
	array = array[:mindex + 1]

	// open tmp file
	file, err := ioutil.TempFile("", "gudgeon-sql-test-*.db")
	if err != nil {
		return err
	}
	file.Close()
	sqlstore.file = file.Name()

	// open db
	db, err := sql.Open("sqlite3", sqlstore.file + "?_sync=OFF&_mutex=NO&_locking=NORMAL&_journal=MEMORY")
	if err != nil {
		return err
	}

	// create table
	stmt, _ := db.Prepare("CREATE TABLE IF NOT EXISTS rules ( rule VARCHAR, groups VARCHAR )")
	stmt.Exec()

	// batch inserts inside one big transaction
	tx, err := db.Begin()

	idx := 0
	end := 0
	for idx < len(array) {
		end = idx + batch_size
		if end > len(array) {
			end = len(array)
		}		
		err = sqlstore.insert(db, tx, array[idx:end])
		if err != nil {
			tx.Rollback()
			return err
		}
		idx += batch_size
	}

	tx.Commit()

	// add index to rule column
	stmt, _ = db.Prepare("CREATE INDEX IF NOT EXISTS rule_idx ON rules (rule)")
	stmt.Exec()

	// force vaccum tables
	stmt, _ = db.Prepare("VACUUM")
	stmt.Exec()

	// close old db
	db.Close()

	// open read-only db
	db, err = sql.Open("sqlite3", sqlstore.file + "?cache=shared&_sync=OFF&_mutex=NO&_locking=NORMAL&_journal=OFF&_query_only=true")
	if err != nil {
		return err
	}
	sqlstore.db = db


	return nil
}

func (sqlstore *sqlstore) Test(forMatch string) (bool, error) {

	if sqlstore.stmt == nil {	
		stmt, err := sqlstore.db.Prepare(searchQuery)
		if err != nil {
			return false, err
		}
		sqlstore.stmt = stmt
	}

	rootform := rootdomain(forMatch)
	resp, err := sqlstore.stmt.Query(forMatch, rootform)
	if err != nil {
		return false, err
	}
	defer resp.Close()

	if resp.Next() {
		return true, nil
	}

	return false, nil
}

func (sqlstore *sqlstore) Teardown() error {
	sqlstore.db.Close()
	os.Remove(sqlstore.file)
	return nil
}