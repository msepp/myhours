package myhours

import (
	"time"
)

const (
	// SettingDefaultCategory is the setting key for default category.
	SettingDefaultCategory = "default_category"
)

// Database defines the database access requirements for stopwatch.
type Database interface {
	// ActiveRecord returns currently active record.
	//
	// If none is active, both return values are nil.
	ActiveRecord() (*Record, error)
	// Records returns all records that fit into the given timespan.
	// Records where starting time is equal or greater to from, and less than before,
	// are returned.
	Records(from, before time.Time) ([]Record, error)
	// RecordsInCategory behaves exactly like Records, but filters also by given
	// categoryID.
	RecordsInCategory(from, before time.Time, categoryID int64) ([]Record, error)
	// ImportRecords with given details. Expects that all records are finished.
	//
	// On success returns the imported record ID.
	ImportRecords(records []Record) ([]int64, error)
	// StartRecord inserts a new active record into the database. If an already active
	// record exist, error is returned instead.
	//
	// On success returns the new record IDs
	StartRecord(start time.Time, categoryID int64, notes string) (int64, error)
	// UpdateRecordCategory to category identified by categoryID for record identified by recordID.
	UpdateRecordCategory(recordID int64, categoryID int64) error
	// FinishRecord identified by recordID by setting the end time details and
	// adding optional notes.
	FinishRecord(recordID int64, start, end time.Time, notes string) error
	// Categories returns all available categories.
	Categories() ([]Category, error)
	// UpdateSetting updates a configuration setting value identified by key.
	UpdateSetting(key string, value string) error
	// Settings returns application settings
	Settings() (*Settings, error)
}
