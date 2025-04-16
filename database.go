package myhours

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

	"github.com/charmbracelet/lipgloss"
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

// ImportRecord allows inserting records into the given database directly without
// going through the application.
//
// Can be used for example to bring in data from other time keeping systems.
func ImportRecord(db *sql.Tx, start time.Time, duration time.Duration, category int, notes string) (int64, error) {
	res, err := db.Exec(`INSERT INTO records ("start", "end", "duration", "category", "notes") VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		start.In(time.UTC).Format(time.RFC3339Nano),
		start.Add(duration).Format(time.RFC3339Nano),
		duration.String(),
		category,
		PtrNonZero(notes),
	)
	if err != nil {
		return 0, fmt.Errorf("db.Exec: %w", err)
	}
	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return 0, fmt.Errorf("db.LastInsertId: %w", err)
	}
	return id, nil
}

func (app Application) startRecord(start time.Time, category int64, notes string) (int64, error) {
	res, err := app.db.Exec(`INSERT INTO records ("start", "category", "notes") VALUES ($1, $2, $3) RETURNING id`,
		start.In(time.UTC).Format(time.RFC3339Nano),
		category,
		notes,
	)
	if err != nil {
		return 0, fmt.Errorf("db.Exec: %w", err)
	}
	var id int64
	if id, err = res.LastInsertId(); err != nil {
		return 0, fmt.Errorf("db.LastInsertId: %w", err)
	}
	return id, nil
}

func (app Application) setRecordCategory(id int64, category int64) error {
	_, err := app.db.Exec(`UPDATE records SET "category"=$2 WHERE "id"=$1`,
		id,
		category,
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (app Application) finishRecord(id int64, start, end time.Time, notes string) error {
	_, err := app.db.Exec(`UPDATE records SET "start"=$2, "end"=$3, "duration"=$4, "notes"=$5 WHERE "id"=$1`,
		id,
		start.In(time.UTC).Format(time.RFC3339Nano),
		end.In(time.UTC).Format(time.RFC3339Nano),
		end.Sub(start).String(),
		PtrNonZero(notes),
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (app Application) partialRecord() (*dbRecord, error) {
	res := app.db.QueryRow(`SELECT "id", "start", "end", "duration", "category", "notes" FROM records WHERE "end" IS NULL ORDER BY id DESC LIMIT 1`)
	record, err := scanDBRecordRow(res)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scanDBRecordRow: %w", err)
	}
	return &record, nil
}

type dbRecord struct {
	ID         int64
	Start      time.Time
	End        time.Time
	Duration   time.Duration
	CategoryID int64
	Notes      string
}

type category struct {
	id           int64
	name         string
	bgColorDark  string
	fgColorDark  string
	bgColorLight string
	fgColorLight string
}

func (c category) ForegroundColor() lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: c.fgColorLight, Dark: c.fgColorDark}
}
func (c category) BackgroundColor() lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: c.bgColorLight, Dark: c.bgColorDark}
}

func (app Application) updateConfig(key string, value string) error {
	_, err := app.db.Exec(`UPDATE configuration SET "value"=$2 WHERE "key"=$1`,
		key,
		value,
	)
	if err != nil {
		return fmt.Errorf("db.Exec: %w", err)
	}
	return nil
}

func (app Application) getCategories() ([]category, error) {
	rows, err := app.db.Query(`SELECT "id", "name", "color_dark_bg", "color_dark_fg", "color_light_bg", "color_light_fg" FROM categories ORDER BY "id" ASC`)
	if err != nil {
		return nil, fmt.Errorf("db.Query: %w", err)
	}
	defer rows.Close()
	var result []category
	for rows.Next() {
		var cat category
		if err = rows.Scan(&cat.id, &cat.name, &cat.bgColorDark, &cat.fgColorDark, &cat.bgColorLight, &cat.fgColorLight); err != nil {
			return nil, fmt.Errorf("rows.Scan: %w", err)
		}
		result = append(result, cat)
	}
	return result, nil
}

func (app Application) getRecords(from, before time.Time) []dbRecord {
	res, err := app.db.Query(
		`SELECT "id", "start", "end", "duration", "category", "notes" FROM records WHERE "start" > $1 AND "end" < $2 ORDER BY start ASC`,
		from.In(time.UTC).Format(time.RFC3339Nano),
		before.In(time.UTC).Format(time.RFC3339Nano),
	)
	if err != nil {
		app.l.Error("failed to query database", slog.String("error", err.Error()))
		return nil
	}
	defer res.Close()
	var records []dbRecord
	for res.Next() {
		var record dbRecord
		if record, err = scanDBRecordRow(res); err != nil {
			err = fmt.Errorf("scanDBRecordRow: %w", err)
			app.l.Error("failed to parse record", slog.String("error", err.Error()))
			continue
		}
		records = append(records, record)
	}
	return records
}

func scanDBRecordRow(row interface{ Scan(dst ...any) error }) (dbRecord, error) {
	var (
		id                   int64
		start                string
		end, duration, notes *string
		categoryID           int64
	)
	if err := row.Scan(&id, &start, &end, &duration, &categoryID, &notes); err != nil {
		return dbRecord{}, fmt.Errorf("row.Scan: %w", err)
	}
	record := dbRecord{ID: id, CategoryID: categoryID, Notes: Val(notes)}
	var err error
	if record.Start, err = time.Parse(time.RFC3339Nano, start); err != nil {
		return dbRecord{}, fmt.Errorf("time.Parse(start): %w", err)
	}
	if end != nil {
		if record.End, err = time.Parse(time.RFC3339Nano, *end); err != nil {
			return dbRecord{}, fmt.Errorf("time.Parse(end): %w", err)
		}
	}
	if duration != nil {
		if record.Duration, err = time.ParseDuration(*duration); err != nil {
			return dbRecord{}, fmt.Errorf("time.ParseDuration(duration): %w", err)
		}
	}
	return record, nil
}

func loadConfig(db *sql.DB, l *slog.Logger) (AppConfig, error) {
	rows, err := db.Query(`SELECT "key", "value" FROM configuration`)
	if err != nil {
		return AppConfig{}, fmt.Errorf("db.Query: %w", err)
	}
	defer rows.Close()
	var config AppConfig
	for rows.Next() {
		var key, value string
		if err = rows.Scan(&key, &value); err != nil {
			return AppConfig{}, fmt.Errorf("rows.Scan: %w", err)
		}
		switch key {
		case "default_category":
			config.DefaultCategory, err = strconv.ParseInt(value, 10, 64)
			if err != nil {
				return AppConfig{}, fmt.Errorf("default_category: strconv.Atoi: %w", err)
			}
		default:
			l.Warn("unsupported configuration key", slog.String("key", key))
		}
	}
	return config, nil
}

type DailyReport struct {
	Date    string
	WeekDay time.Weekday
	Month   time.Month
	Total   time.Duration
	Notes   []string
}

type WeeklyReport struct {
	Year   int
	WeekNo int
	Total  time.Duration
	Days   []DailyReport
}

func (r WeeklyReport) DateRange() (string, string) {
	if len(r.Days) == 0 {
		return "", ""
	}
	if len(r.Days) == 1 {
		return r.Days[0].Date, r.Days[0].Date
	}
	return r.Days[0].Date, r.Days[len(r.Days)-1].Date
}

type MonthlyReport struct {
	Year      int
	Month     time.Month
	FirstDate string
	LastDate  string
	Total     time.Duration
	Weeks     []WeeklyReport
}

func (r MonthlyReport) DateRange() (string, string) {
	if len(r.Weeks) == 0 {
		return "", ""
	}
	if len(r.Weeks) == 1 {
		return r.Weeks[0].DateRange()
	}
	first, _ := r.Weeks[0].DateRange()
	_, last := r.Weeks[len(r.Weeks)-1].DateRange()
	return first, last
}

type YearlyReport struct {
	Year   int
	Months []MonthlyReport
	Total  time.Duration
}

func (r YearlyReport) DateRange() (string, string) {
	if len(r.Months) == 0 {
		return "", ""
	}
	if len(r.Months) == 1 {
		return r.Months[0].DateRange()
	}
	first, _ := r.Months[0].DateRange()
	_, last := r.Months[len(r.Months)-1].DateRange()
	return first, last
}

func recordsAsWeeks(records []dbRecord) []WeeklyReport {
	var (
		weeks []WeeklyReport
		cw    *WeeklyReport
	)
	for _, record := range records {
		// show dates in local time. They are stored in UTC.
		start := record.Start.In(time.Local)
		y, weekNo := start.In(time.Local).ISOWeek()
		wd := int(start.Weekday())
		if wd == 0 { // Sunday is zero. Horrible.
			wd = 7
		}
		wd--
		if cw == nil || cw.Year != y || cw.WeekNo != weekNo {
			weeks = append(weeks, WeeklyReport{Year: y, WeekNo: weekNo})
			cw = &weeks[len(weeks)-1]
			// seed the days of the week to get a full week.
			for i := range 7 {
				d := start.AddDate(0, 0, -1*(wd-i))
				cw.Days = append(cw.Days, DailyReport{
					Date:    d.Format(time.DateOnly),
					WeekDay: d.Weekday(),
					Month:   d.Month(),
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

func recordsAsMonths(records []dbRecord) []MonthlyReport {
	var (
		months []MonthlyReport
		cm     *MonthlyReport
		cw     *WeeklyReport
	)
	for _, r := range records {
		start := r.Start.In(time.Local)
		y, m, _ := start.Date()
		if cm == nil || cm.Month != m || cm.Year != y {
			first := time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
			last := time.Date(y, m+1, -1, 0, 0, 0, 0, time.Local)
			months = append(months, MonthlyReport{
				Year:      y,
				Month:     m,
				FirstDate: first.Format(time.DateOnly),
				LastDate:  last.Format(time.DateOnly),
			})
			cm = &months[len(months)-1]
			// create the month weeks
			prevWeekNo := 0
			for !first.After(last) {
				if _, weekNo := first.ISOWeek(); prevWeekNo < weekNo {
					prevWeekNo = weekNo
					wr := WeeklyReport{
						Year:   y,
						WeekNo: weekNo,
					}
					// seed the days of the week to get a full week.
					dd := first
					_, wNo := dd.ISOWeek()
					for wNo == weekNo && dd.Month() == m {
						wr.Days = append(wr.Days, DailyReport{
							Date:    dd.Format(time.DateOnly),
							WeekDay: dd.Weekday(),
							Month:   dd.Month(),
						})
						dd = dd.AddDate(0, 0, 1)
						_, wNo = dd.ISOWeek()
					}
					cm.Weeks = append(cm.Weeks, wr)
				}
				first = first.AddDate(0, 0, 1)
			}
			cw = nil
		}
		_, weekNo := r.Start.In(time.Local).ISOWeek()
		if cw == nil || cw.WeekNo != weekNo {
			for i, w := range cm.Weeks {
				if w.WeekNo == weekNo {
					cw = &cm.Weeks[i]
				}
			}
			if cw == nil {
				panic("week not set somehow!")
			}
		}
		cm.Total += r.Duration
		cw.Total += r.Duration
		date := start.Format(time.DateOnly)
		for i := range cw.Days {
			if cw.Days[i].Date != date {
				continue
			}
			cw.Days[i].Total += r.Duration
			break
		}
		wd := start.Weekday()
		if wd == 0 {
			wd = 7
		}
		wd--
		cw.Days[wd].Total += r.Duration
		if r.Notes != "" {
			cw.Days[wd].Notes = append(cw.Days[wd].Notes, r.Notes)
		}
	}
	return months
}

func recordsAsYears(records []dbRecord) []YearlyReport {
	var (
		years []YearlyReport
		cy    *YearlyReport
	)
	for _, m := range recordsAsMonths(records) {
		if cy == nil || cy.Year != m.Year {
			years = append(years, YearlyReport{Year: m.Year})
			cy = &years[len(years)-1]
			for i := range 12 {
				cy.Months = append(cy.Months, MonthlyReport{
					Year:  m.Year,
					Month: time.Month(i + 1),
				})
			}
		}
		cy.Months[m.Month-1] = m
		cy.Total += m.Total
	}
	return years
}
