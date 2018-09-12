package database

import (
    //"errors"
	//"fmt"
	"bufio"
 	"database/sql"
 	"io"
 	"strconv"
 	"strings"
 	_ "github.com/mattn/go-sqlite3"
)

const (
	ListTypeWhite = 0
	ListTypeBlack = 1
	ListTypeBlock = 2

	RuleTypeMatch = 0
	RuleTypeWild = 1
	RuleTypeRegex = 2
)

var db_init = []string{
	"CREATE TABLE IF NOT EXISTS `rule` ( `block` VARCHAR, `type` integer, `list` VARCHAR, `list_type` integer, `tag` VARCHAR , `active` bool DEFAULT false, PRIMARY KEY(block, type, list, list_type, tag, active));",
	"CREATE TABLE IF NOT EXISTS `cache` (`question` VARCHAR, `domain` VARCHAR, `answer` VARCHAR, `ttl` integer, `created` DATETIME DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY(question, domain));",
}

type gudgeonDatabaseImpl struct {
	dbPath string
	dbHandle *sql.DB
}

type GudgeonDatabase interface {
	CheckCache(question string, domain string) (string, error)
	InsertCache(question string, domain string, answer string, ttl int) error
	InsertInactiveRule(block string, blockType int, fromListName string, fromListType int, tags []string) error
	InsertRule(block string, blockType int, fromListName string, fromListType int, tags []string, active bool) error
	BulkInsertRule(reader io.Reader, fromListName string, fromListType int, tags []string, active bool) error
	GetRules(domain string, listType int, listNames []string, tags []string) error
	Close() error
}

func Get(pathSpecification string) (GudgeonDatabase, error) {
	// create new db instance
	db := new(gudgeonDatabaseImpl)
	db.dbPath = pathSpecification
	err := db.init()
	return db, err
}

// lazy init db connection handle
func (db *gudgeonDatabaseImpl) open() (*sql.DB, error) {
	if db.dbHandle == nil {
		var err error
		db.dbHandle, err = sql.Open("sqlite3", db.dbPath)
		if err != nil {
			return nil, err
		}
	}
	return db.dbHandle, nil
}

func (db *gudgeonDatabaseImpl) init() error {
	// open db path and set handle
	handle, dbOpenErr := db.open()
	if dbOpenErr != nil {
		return dbOpenErr
	}

	// set only one connection
	handle.SetMaxOpenConns(5)

	// delete tables and create new tables
	// we can do this because a database
	// shouldn't be opened and then be useful
	// because it's mostly runtime-oriented
	// and not run-to-run
	for _, statement := range db_init {
		stmt, err := handle.Prepare(statement)
		if err != nil {
			return err
		}
		_, err = stmt.Exec()
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *gudgeonDatabaseImpl) CheckCache(question string, domain string) (string, error) {
	handle, err := db.open()
	if err != nil {
		return "", err
	}

	stmt, err := handle.Prepare("SELECT answer FROM cache WHERE question=? and domain=? LIMIT 1;")
	if err != nil {
		return "", err
	}

	answer := ""
	resp, err := stmt.Query(question, domain)
	if err != nil {
		return "", err
	}
	if resp.Next() {
		err = resp.Scan(&answer)
		if err != nil {
			resp.Close()
			return answer, err
		}
	}
	resp.Close()

	// return answer
	return answer, nil
}

func (db *gudgeonDatabaseImpl) InsertCache(question string, domain string, answer string, ttl int) error {
	handle, err := db.open()
	if err != nil {
		return err
	}

	stmt, err := handle.Prepare("INSERT OR REPLACE INTO cache (question, domain, answer, ttl) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(question, domain, answer, ttl)
	if err != nil {
		return err
	}

	return nil
}

func (db *gudgeonDatabaseImpl) insertTaggedRule(block string, blockType int, fromListName string, fromListType int, tag string, active bool) error {
	handle, err := db.open()
	if err != nil {
		return err
	}

	stmt, err := handle.Prepare("INSERT OR REPLACE INTO rule (block, type, list, list_type, tag, active) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return err
	}

	_, err = stmt.Exec(block, blockType, fromListName, fromListType, tag, active)
	if err != nil {
		return err
	}

	return nil
}

func (db *gudgeonDatabaseImpl) InsertInactiveRule(block string, blockType int, fromListName string, fromListType int, tags []string) error {
	return db.InsertRule(block, blockType, fromListName, fromListType, tags, false)
}

func (db *gudgeonDatabaseImpl) InsertRule(block string, blockType int, fromListName string, fromListType int, tags []string, active bool) error {

	if len(tags) == 0 {
		tags = []string{""}
	}
	
	for _, tag := range tags {
		err := db.insertTaggedRule(block, blockType, fromListName, fromListType, tag, active)
		if err != nil {
			return err
		}
	}	

	return nil
}

func (db *gudgeonDatabaseImpl) BulkInsertRule(reader io.Reader, fromListName string, fromListType int, tags []string, active bool) error {

	var builder strings.Builder
	builder.WriteString("INSERT OR REPLACE INTO rule (block, type, list, list_type, tag, active) VALUES")

	// used as a cheap comman-based join mechanism
	firstRecord := true

	// no tags means one empty tag
	if len(tags) == 0 {
		tags = []string{""}
	}

	// from list name one time conv
	fromListTypeInt := strconv.Itoa(fromListType)

	// active status
	activeStr := "0"
	if active {
		activeStr = "1"
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		// get domain name text from reader
		text := scanner.Text()

		// write each tag
		for _, tag := range tags {
			// do cheap join
			if !firstRecord {
				builder.WriteString(", ")
			}
			firstRecord = false

			// write record
			builder.WriteString("(")
			// block string
			builder.WriteString("\"")
			builder.WriteString(text)
			builder.WriteString("\", ")
			// block type, todo: needs to be infered from the actual rule
			builder.WriteString("0")
			builder.WriteString(", ")
			// list name
			builder.WriteString("\"")
			builder.WriteString(fromListName)
			builder.WriteString("\", ")
			// list type
			builder.WriteString(fromListTypeInt)
			builder.WriteString(", ")
			// tag
			builder.WriteString("\"")
			builder.WriteString(tag)
			builder.WriteString("\", ")
			// active status
			builder.WriteString(activeStr)
			// end record
			builder.WriteString(")")
		}

	}
	// write end of statement
	builder.WriteString(";")

	// open/get handle
	handle, err := db.open()
	if err != nil {
		return err
	}

	// create statement
	stmt, err := handle.Prepare(builder.String())
	if err != nil {
		return err
	}

	// simple (no parameter) exec
	_, err = stmt.Exec()
	if err != nil {
		return err
	}

	return nil
}

// 
/*
// list-based query for type 0-1 (white/black lists with more potentially more intensive processing)
select distinct i.block, l.name, l.type from rule i
join list_join lj on i.id=lj.list_item_id
join (select * from list l where l.name in ('standard', 'default', 'none') and l.type<2) l on l.id=lj.list_id

// list-based query for type 2 (blocklists)
select distinct i.block, l.name, l.type from rule i
join list_join lj on i.id=lj.list_item_id
join (select * from list l where l.name in ('standard', 'default', 'none') and l.type=2) l on l.id=lj.list_id
where i.block = 'advert.com'

// tag-based query for type 0-1 blocklists
select distinct i.block, l.type, t.name from rule i
join tag_join tj on i.id=tj.list_item_id
join (select * from tag t where t.name in ('advert','none')) t on tj.tag_id=t.id
join list_join lj on i.id=lj.list_item_id
join list l on l.id=lj.list_id and l.type < 2

// tag-based query for type 2 blocklists
select distinct i.block, l.type, t.name from rule i
join tag_join tj on i.id=tj.list_item_id
join (select * from tag t where t.name in ('advert','none')) t on tj.tag_id=t.id
join list_join lj on i.id=lj.list_item_id
join list l on l.id=lj.list_id and l.type = 2
where i.block = 'advert.com'
*/
func (db *gudgeonDatabaseImpl) GetRules(domain string, listType int, listNames []string, tags []string) error {
	_, dbOpenErr := db.open()
	if dbOpenErr != nil {
		return dbOpenErr
	}



	return nil
}

func (db *gudgeonDatabaseImpl) Close() error {
	err := db.dbHandle.Close()
	db.dbHandle = nil
	return err
}