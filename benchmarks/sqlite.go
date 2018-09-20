package benchmarks

import (
	"bufio"
  	"database/sql"
 	"io/ioutil"
 	"os"
 	"strings"

    _ "github.com/mattn/go-sqlite3"
)

type sqlstore struct {
	db *sql.DB
}

func (sqlstore *sqlstore) insert(tx *sql.Tx, rule string) {
	stmt, _ := sqlstore.db.Prepare("INSERT INTO rules (rule) VALUES (?)")

	tx.Stmt(stmt).Exec(rule)
}

func (sqlstore *sqlstore) Load(inputfile string) error {
	// go through file
	data, err := os.Open(inputfile)
	if err != nil {
		return err
	}
	defer data.Close()

	// open tmp dir
	file, err := ioutil.TempFile("", "gudgeon-sql-*.db")
	if err != nil {
		return err
	}
	defer file.Close()

	// open db
	db, err := sql.Open("sqlite3", file.Name())
	if err != nil {
		return err
	}

	// create table
	stmt, _ := db.Prepare("CREATE TABLE IF NOT EXISTS rules ( rule VARCHAR PRIMARY KEY );")
	stmt.Exec()

	// save db
	sqlstore.db = db

	tx, _ := db.Begin()
	scanner := bufio.NewScanner(data)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if "" == text {
			continue
		}
		sqlstore.insert(tx, text)
	}
	tx.Commit()

	return nil
}


func (sqlstore *sqlstore) Test(forMatch string) (bool, error) {
	rootform := rootdomain(forMatch)
	stmt, _ := sqlstore.db.Prepare("SELECT count(*) as count FROM rules WHERE rule = ? OR rule = ?")
	resp, _ := stmt.Query(forMatch, rootform)
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