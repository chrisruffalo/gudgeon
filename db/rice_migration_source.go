package db

import (
    "fmt"
    "bytes"
    "os"
    "sort"
    "strings"

    rice "github.com/GeertJohan/go.rice"
    migrate "github.com/rubenv/sql-migrate"
)

type RiceMigrationSource struct {
    box *rice.Box
}

func NewMigrationSource(box *rice.Box) *RiceMigrationSource {
    return &RiceMigrationSource{box: box}
}

// also needed the same sort implementation
type byId []*migrate.Migration
func (b byId) Len() int           { return len(b) }
func (b byId) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b byId) Less(i, j int) bool { return b[i].Less(b[j]) }

// https://github.com/rubenv/sql-migrate/blob/master/migrate.go for inspiration from the packr implementation
func (source *RiceMigrationSource) FindMigrations() ([]*migrate.Migration, error) {
    migrations := make([]*migrate.Migration, 0)

    // walk the sources in the box
    source.box.Walk("/", func(path string, info os.FileInfo, err error) error {
        // empty path is nothing
        if "" == path || "" == strings.TrimSpace(path) {
            return nil
        }

        // must end with ".sql"
        if !strings.HasSuffix(strings.ToLower(path), ".sql") {
            return nil
        }

        // use the migration stuff to parse the migration from the rice box
        migrationBytes, err := source.box.Bytes(path)
        if err != nil {
            return nil
        }

        migration, err := migrate.ParseMigration(path, bytes.NewReader(migrationBytes))
        if err != nil {
            fmt.Printf("Failed to parse migration for file '%s' with error: %s\n", path, err)
            return nil
        }

        // add migrations to list
        migrations = append(migrations, migration)

        return nil
    })

    // now sort
    sort.Sort(byId(migrations))

    return migrations, nil
}
