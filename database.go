package myhours

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
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

func (app Application) insertRecord(start, end time.Time, category int, notes string) error {
	_, err := app.db.Exec(`INSERT INTO records ("start", "end", "duration", "category", "notes") VALUES ($1, $2, $3, $4, $5)`,
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

type dbRecord struct {
	Start    time.Time
	End      time.Time
	Duration time.Duration
	Category int
	Notes    string
}

func (app Application) getRecords() []dbRecord {
	res, err := app.db.Query(`SELECT "start", "end", "duration", "notes" FROM records`)
	if err != nil {
		app.l.Error("failed to query database", slog.String("error", err.Error()))
		return nil
	}
	var records []dbRecord
	for res.Next() {
		var start, end, duration, notes string
		if err = res.Scan(&start, &end, &duration, &notes); err != nil {
			app.l.Error("failed to scan", slog.String("error", err.Error()))
			return nil
		}
		record := dbRecord{Notes: notes}
		if record.Start, err = time.Parse(time.RFC3339Nano, start); err != nil {
			app.l.Error("failed to parse record start time, skipping", slog.String("error", err.Error()))
			continue
		}
		if record.End, err = time.Parse(time.RFC3339Nano, end); err != nil {
			app.l.Error("failed to parse record end time, skipping", slog.String("error", err.Error()))
			continue
		}
		if record.Duration, err = time.ParseDuration(duration); err != nil {
			app.l.Error("failed to parse record duration, skipping", slog.String("error", err.Error()))
			continue
		}
		records = append(records, record)
	}
	return records
}

type day struct {
	Date    string
	WeekDay time.Weekday
	Total   time.Duration
	Notes   []string
}

type week struct {
	Year   int
	WeekNo int
	Days   []day
	Total  time.Duration
}

func (w week) ISOWeekString() string {
	return fmt.Sprintf("%04dw%02d", w.Year, w.WeekNo)
}

func recordsAsWeeks(records []dbRecord) []week {
	var (
		weeks []week
		cw    *week
	)
	for _, record := range records {
		// show dates in local time. They are stored in UTC.
		start := record.Start.In(time.Local)
		year, weekNo := start.In(time.Local).ISOWeek()
		wd := int(start.Weekday())
		if wd == 0 { // Sunday is zero. Horrible.
			wd = 7
		}
		wd--
		if cw == nil || cw.Year != year || cw.WeekNo != weekNo {
			weeks = append(weeks, week{Year: year, WeekNo: weekNo})
			cw = &weeks[len(weeks)-1]
			// seed the days of the week to get a full week.
			for i := range 7 {
				d := start.AddDate(0, 0, -1*(wd-i))
				cw.Days = append(cw.Days, day{
					Date:    d.Format(time.DateOnly),
					WeekDay: d.Weekday(),
				})
			}
		}
		cw.Total += record.Duration
		cd := &cw.Days[wd]
		cd.Total += record.Duration
		if record.Notes != "" {
			cd.Notes = append(cd.Notes, record.Notes)
		}
	}
	return weeks
}
