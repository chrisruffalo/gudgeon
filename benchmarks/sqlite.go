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
	// mut be less than 1000 and a multiple of insertValueSize
	batch_size = 999

	insertStmt = "INSERT INTO rules (rule) VALUES (?)"
	insertValues = "(?)"
	insertValueSize = 1

	stmt = "SELECT rule FROM rules WHERE rule = ? OR rule = ? LIMIT 1;"
)

func (sqlstore *sqlstore) Id() string {
	return "sql: '" + sqlstore.file + "'"
}

func (sqlstore *sqlstore) insert(db *sql.DB, rules []string) error {
	tx, _ := db.Begin()

	// build value-statement
	if sqlstore.insertCacheStmtSize != len(rules) {
		s := insertStmt + strings.Repeat(", " + insertValues, (len(rules) - 1)/insertValueSize)

		stmt, err := db.Prepare(s)
		if err != nil {
			return err
		}

		// cache statement and statement size
		sqlstore.insertCacheStmt = stmt
		sqlstore.insertCacheStmtSize = len(rules)
	}
	
	args := make([]interface{}, len(rules))
	for idx, _ := range rules {
		args[idx] = rules[idx]
	}

	_, err := tx.Stmt(sqlstore.insertCacheStmt).Exec(args...)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()

	return nil
}

func (sqlstore *sqlstore) Load(inputfile string) error {
	// go through file
	content, err := ioutil.ReadFile(inputfile)
	if err != nil {
		return err
	}
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
	stmt, _ := db.Prepare("CREATE TABLE IF NOT EXISTS rules ( rule VARCHAR PRIMARY KEY );")
	stmt.Exec()

	// batch inserts
	idx := 0
	end := 0
	for idx < len(array) {
		end += batch_size
		if end > len(array) {
			end = len(array)
		}		
		err = sqlstore.insert(db, array[idx:end])
		if err != nil {
			return err
		}
		idx += batch_size
	}

	// force vaccum tables
	stmt, _ = db.Prepare("VACUUM;")
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
		stmt, err := sqlstore.db.Prepare(stmt)
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