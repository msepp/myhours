package myhours

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

//go:embed schema/default.db
var defaultDB []byte

// NewSQLiteDatabase opens a SQLite database from given location.
//
// If no database exists in the given location, new database is initialized and
// opened instead.
func NewSQLiteDatabase(dbFile string) (*sql.DB, error) {
	// Make sure the database location (directory) exists. The operation should
	// return no errors if directory already exists.
	dbDir := filepath.Dir(dbFile)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("create data directory: os.MkdirAll: %w", err)
	}
	// If the database file does not exist, we copy the default embedded database
	// as starting point.
	if _, err := os.Stat(dbFile); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("check if database exists: os.Stat: %w", err)
		}
		// create the database file and copy embedded contents into it.
		if err = os.WriteFile(dbFile, defaultDB, 0644); err != nil {
			return nil, fmt.Errorf("initialize default database: os.WriteFile: %w", err)
		}
	}
	// Nice, we have a location for the database. Try to open it, and check if
	// there's a need to init or migrate schema.
	db, err := sql.Open("sqlite", dbFile+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

func insertRecord(db *sql.DB, start, end time.Time, category int, notes string) error {
	_, err := db.Exec("INSERT INTO records (start, end, duration, category, notes) VALUES ($1, $2, $3, $4, $5)",
		start.In(time.UTC).Format(time.RFC3339Nano),
		end.In(time.UTC).Format(time.RFC3339Nano),
		end.Sub(start).String(),
		category,
		notes)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}
