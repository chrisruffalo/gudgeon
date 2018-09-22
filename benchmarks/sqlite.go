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
}

const (
	batch_size = 800
)

func (sqlstore *sqlstore) Id() string {
	return "sql: '" + sqlstore.file + "'"
}

func (sqlstore *sqlstore) insert(db *sql.DB, rules []string) error {
	tx, _ := db.Begin()

	// build value-statement
	s := "INSERT INTO rules (rule) VALUES (?)" + strings.Repeat(", (?)", len(rules) - 1)

	stmt, err := db.Prepare(s)
	if err != nil {
		tx.Rollback()
		return err
	}
	
	args := make([]interface{}, len(rules))
	for idx, _ := range rules {
		args[idx] = rules[idx]
	}

	_, err = tx.Stmt(stmt).Exec(args...)
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
	array = array[:mindex+1]

	// open tmp dir
	file, err := ioutil.TempFile("", "gudgeon-sql-test-*.db?_sync=OFF&_mutex=NO&_locking=NORMAL&_journal=MEMORY")
	if err != nil {
		return err
	}
	file.Close()
	sqlstore.file = file.Name()

	// open db
	db, err := sql.Open("sqlite3", sqlstore.file)
	if err != nil {
		return err
	}
	sqlstore.db = db

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

	stmt, _ = db.Prepare("VACUUM;")
	stmt.Exec()

	return nil
}

func (sqlstore *sqlstore) Test(forMatch string) (bool, error) {
	db := sqlstore.db

	stmt, err := db.Prepare("SELECT count(*) as count FROM rules WHERE rule = ? OR rule = ?")
	if err != nil {
		return false, err
	}

	rootform := rootdomain(forMatch)
	resp, err := stmt.Query(forMatch, rootform)
	if err != nil {
		return false, err
	}
	defer resp.Close()

	count := 0
	if resp.Next() {
		err := resp.Scan(&count)
		if err != nil {
			return false, err
		}

		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}

func (sqlstore *sqlstore) Teardown() error {
	sqlstore.db.Close()
	os.Remove(sqlstore.file)
	return nil
}