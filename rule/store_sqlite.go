package rule

import (
    "database/sql"
    "fmt"
    "os"
    "path"
    "strings"

    _ "github.com/mattn/go-sqlite3"

    "github.com/chrisruffalo/gudgeon/config"
    "github.com/chrisruffalo/gudgeon/util"
)

type sqlStore struct {
    db       *sql.DB
}

func (store *sqlStore) Load(conf *config.GudgeonConfig, list *config.GudgeonList, sessionRoot string, rules []Rule) uint64 {
    // create new database if it doesn't exist
    if store.db == nil {
        // get session storage location
        sessionDb := path.Join(sessionRoot, "rules.db")
        if _, err := os.Stat(sessionRoot); os.IsNotExist(err) {
            os.MkdirAll(sessionRoot, os.ModePerm)
        }
        db, err := sql.Open("sqlite3", sessionDb + "?_sync=FULL")
        if err != nil {
            fmt.Printf("error: %s\n", err)
            return 0
        }
        store.db = db
    }

    // make the database structure for the given list
    listName := list.ShortName()

    stmt := "CREATE TABLE IF NOT EXISTS " + listName + " ( Rule TEXT );"
    _, err := store.db.Exec(stmt)
    if err != nil {
        fmt.Printf("error: %s\n", err)
        return 0
    }

    // filter through rules and count how many rules are in use
    counter := uint64(0)

    // insert rules
    currentIndex := 0
    batchSize := 250
    startStmt := "INSERT INTO " + listName + " (Rule) VALUES"
    maxStmt := startStmt + " (?)" + strings.Repeat(", (?)", batchSize - 1) 
    ruleFaces := make([]interface{}, batchSize)

    for currentIndex < len(rules) {
        batchEndIdx := currentIndex + batchSize - 1
        if batchEndIdx >= len(rules) {
            batchEndIdx = len(rules) - 1
        }
        ctr := 0

        for idx := currentIndex; idx <= batchEndIdx; idx++ {
            if rules[idx] == nil {
                continue
            }
            if "" == rules[idx].Text() {
                continue
            }
            ruleFaces[ctr] = rules[idx].Text()
            ctr++
        }

        // if there are rows to insert
        if ctr > 0 {
            _, err := store.db.Exec(maxStmt[:len(startStmt) - 1 + len(ruleFaces) * 5], ruleFaces...)
            if err != nil {
                fmt.Printf("error during insert: %s\n", err)
            } else {
                counter += uint64(ctr)
            }
        }

        currentIndex = currentIndex + batchSize
        // reset rule faces
        for idx, _ := range ruleFaces {
            ruleFaces[idx] = nil
        }
    }

    // after everything is inserted, add index in one go
    idxStmt := "CREATE INDEX IF NOT EXISTS idx_" + listName + "_Rules ON " + listName + " (Rule);"
    _, err = store.db.Exec(idxStmt)
    if err != nil {
        fmt.Printf("Could not create index on table %s", listName)
    }

    // return rule count
    return counter
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
        params = fmt.Sprintf("%s$%d", params, idx + 1)
        dfaces[idx] = domain
    }

    stmt := "SELECT Rule FROM " + listName + " WHERE Rule in ( " + params + " ) LIMIT 1"
    pstmt, err := store.db.Prepare(stmt)
    if err != nil {
        fmt.Printf("err: %s\n", err)
    }

    rows, err := pstmt.Query(dfaces...)
    defer rows.Close()
    if err != nil {
        fmt.Printf("err: %s\n", err)
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