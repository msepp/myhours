// Package sqlite implements myhours.Database on top of SQLite.
//
// Contains also utilities for initializing a new SQLite databases with the required
// schema.
package sqlite

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/glebarez/go-sqlite"
	"github.com/msepp/myhours"
)

//go:embed default.db
var defaultDB []byte

const (
	selectFullRecord       = `SELECT "id", "start", "end", "duration", "category", "notes" FROM records`
	queryActiveRecord      = selectFullRecord + ` WHERE "end" IS NULL ORDER BY id DESC LIMIT 1`
	queryRecords           = selectFullRecord + ` WHERE "start" > $1 AND "end" < $2 ORDER BY start ASC`
	queryRecordsOfCategory = selectFullRecord + ` WHERE "start" > $1 AND "end" < $2 AND "category" = $3 ORDER BY start ASC`
	queryCategories        = `SELECT "id", "name", "color_dark_bg", "color_dark_fg", "color_light_bg", "color_light_fg" FROM categories ORDER BY "id" ASC`
	insertFullRecord       = `INSERT INTO records ("start", "end", "duration", "category", "notes") VALUES ($1, $2, $3, $4, $5) RETURNING id`
	insertActiveRecord     = `INSERT INTO records ("start", "category", "notes") VALUES ($1, $2, $3) RETURNING id`
	updateRecordCategory   = `UPDATE records SET "category" = $2 WHERE "id" = $1`
	updateRecordAsFinished = `UPDATE records SET "start" = $2, "end" = $3, "duration" = $4, "notes" = $5 WHERE "id" = $1`
	queryConfigSettings    = `SELECT "key", "value" FROM configuration`
	updateConfigSetting    = `UPDATE configuration SET "value" = $2 WHERE "key" = $1`
)

// SQLite implements Database on top of SQLite.
type SQLite struct {
	db *sql.DB
	l  *slog.Logger
}

// ActiveRecord implements myhours.Database ActiveRecord on top of SQLite
func (db *SQLite) ActiveRecord() (*myhours.Record, error) {
	res := db.db.QueryRow(queryActiveRecord)
	record, err := scanRecord(res)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanDBRecordRow: %w", err)
	}
	return record, nil
}

// Records implements myhours.Database Records on top of SQLite
func (db *SQLite) Records(from, before time.Time) ([]myhours.Record, error) {
	res, err := db.db.Query(queryRecords, from.In(time.UTC).Format(time.RFC3339Nano), before.In(time.UTC).Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() { _ = res.Close() }()
	// scan all rows into dbRecords
	var records []myhours.Record
	for res.Next() {
		var record *myhours.Record
		if record, err = scanRecord(res); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, *record)
	}
	return records, nil
}

// RecordsInCategory implements myhours.Database RecordsInCategory on top of SQLite
func (db *SQLite) RecordsInCategory(from, before time.Time, categoryID int64) ([]myhours.Record, error) {
	res, err := db.db.Query(queryRecordsOfCategory, from.In(time.UTC).Format(time.RFC3339Nano), before.In(time.UTC).Format(time.RFC3339Nano), categoryID)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() { _ = res.Close() }()
	// scan all rows into dbRecords
	var records []myhours.Record
	for res.Next() {
		var record *myhours.Record
		if record, err = scanRecord(res); err != nil {
			return nil, fmt.Errorf("scan record: %w", err)
		}
		records = append(records, *record)
	}
	return records, nil
}

// ImportRecords implements Database.ImportRecords on top of SQLite.
//
// Inserts are done in a transaction, so the result is all or nothing.
//
// Returns the IDs of created records.
func (db *SQLite) ImportRecords(records []myhours.Record) ([]int64, error) {
	// first validate all records
	for _, record := range records {
		if !record.Finished() {
			return nil, errors.New("all records must be finished")
		}
		if err := record.Validate(); err != nil {
			return nil, fmt.Errorf("validate record: %w", err)
		}
	}
	// Then import in a transaction
	tx, err := db.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	var results []int64
	for _, record := range records {
		var res sql.Result
		if res, err = tx.Exec(insertFullRecord,
			record.Start.In(time.UTC).Format(time.RFC3339Nano),
			record.End.Format(time.RFC3339Nano),
			record.Duration.String(),
			record.CategoryID,
			ptrNonZero(record.Notes),
		); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				db.l.Warn("failed to rollback transaction", slog.String("error", rollbackErr.Error()))
			}
			return nil, fmt.Errorf("db.Exec: %w", err)
		}
		var id int64
		if id, err = res.LastInsertId(); err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				db.l.Warn("failed to rollback transaction", slog.String("error", rollbackErr.Error()))
			}
			return nil, fmt.Errorf("db.LastInsertId: %w", err)
		}
		results = append(results, id)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}
	return results, nil
}

func (db *SQLite) StartRecord(start time.Time, categoryID int64, notes string) (int64, error) {
	active, err := db.ActiveRecord()
	if err != nil {
		return 0, fmt.Errorf("active record: %w", err)
	}
	if active != nil {
		return 0, errors.New("active record already exists")
	}
	var res sql.Result
	if res, err = db.db.Exec(insertActiveRecord, start.In(time.UTC).Format(time.RFC3339Nano), categoryID, notes); err != nil {
		return 0, fmt.Errorf("db.Exec: %w", err)
	}
	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return 0, fmt.Errorf("db.LastInsertId: %w", err)
	}
	return id, nil
}

func (db *SQLite) UpdateRecordCategory(recordID int64, categoryID int64) error {
	if _, err := db.db.Exec(updateRecordCategory, recordID, categoryID); err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (db *SQLite) FinishRecord(recordID int64, start, end time.Time, notes string) error {
	_, err := db.db.Exec(updateRecordAsFinished,
		recordID,
		start.In(time.UTC).Format(time.RFC3339Nano),
		end.In(time.UTC).Format(time.RFC3339Nano),
		end.Sub(start).String(),
		ptrNonZero(notes),
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (db *SQLite) Categories() ([]myhours.Category, error) {
	rows, err := db.db.Query(queryCategories)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []myhours.Category
	for rows.Next() {
		var cat myhours.Category
		if err = rows.Scan(&cat.ID, &cat.Name, &cat.BackgroundDark, &cat.ForegroundDark, &cat.BackgroundLight, &cat.ForegroundLight); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		result = append(result, cat)
	}
	return result, nil
}

func (db *SQLite) UpdateSetting(key string, value string) error {
	if _, err := db.db.Exec(updateConfigSetting, key, value); err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (db *SQLite) Settings() (*myhours.Settings, error) {
	rows, err := db.db.Query(queryConfigSettings)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var config myhours.Settings
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		switch key {
		case myhours.SettingDefaultCategory:
			config.DefaultCategoryID, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf(key+": strconv.Atoi: %w", err)
			}
		default:
			db.l.Warn("unsupported configuration key", slog.String("key", key))
		}
	}
	return &config, nil
}

// Option defines option function for SQLite based Database solution.
type Option func(*SQLite)

// Logger sets the logger for the SQLite Database implementation.
func Logger(l *slog.Logger) Option {
	return func(db *SQLite) {
		db.l = l
	}
}

// InitiateSQLiteDatabase opens or creates an SQLite database to given destination.
//
// If no database exists in the given location, new database is initialized and
// opened instead. Creates directories as needed.
func InitiateSQLiteDatabase(dbFile string) (*sql.DB, error) {
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

// NewSQLite return a SQLite database implementation using given *sql.DB handle.
//
// Use for example InitiateSQLiteDatabase to open/initiate an SQLite based *sql.DB handle.
func NewSQLite(handle *sql.DB, options ...Option) *SQLite {
	db := &SQLite{db: handle, l: slog.New(slog.DiscardHandler)}
	for _, option := range options {
		option(db)
	}
	return db
}
