package main

import (
	"bufio"
	"database/sql"
	_ "embed"
	"flag"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/msepp/myhours"

	_ "github.com/glebarez/go-sqlite"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

type importedRecord struct {
	start    time.Time
	duration time.Duration
	category int
	notes    string
}

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("failed to determine user config directory", slog.String("error", err.Error()))
	}
	var (
		doImport   bool
		importFile = "import.txt"
		dbLocation = path.Join(configDir, "my-hours-cli")
		dbFile     = path.Join(dbLocation, "database.db")
	)
	flag.StringVar(&dbFile, "db", dbFile, "Database location")
	flag.BoolVar(&doImport, "import", doImport, "Run data import. -importFile selects import data location.")
	flag.StringVar(&importFile, "importFile", importFile, "File with import data. Must contain lines in format '2006-01-02T15:04:05.999999999Z07:00,<duration>,categoryInt,notes'. Notes can not contain newlines.")
	flag.Parse()

	logger.Info("opening database", slog.String("database", dbFile))
	// Initialize the database
	var db *sql.DB
	if db, err = myhours.NewSQLiteDatabase(dbFile); err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	// If user wants to do import, do it now that we know we have a destination
	// database ready.
	if doImport {
		logger.Info("running import", slog.String("from", importFile), slog.String("to", dbFile))
		var f *os.File
		if f, err = os.Open(importFile); err != nil {
			logger.Error("reading import file failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		scanner := bufio.NewScanner(f)
		var entries []importedRecord
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			pcs := strings.SplitN(line, ",", 4)
			if len(pcs) != 4 {
				logger.Error("invalid line format", slog.String("line", line))
				os.Exit(2)
			}
			var entry importedRecord
			if entry.start, err = time.Parse(time.RFC3339Nano, pcs[0]); err != nil {
				logger.Error("invalid start time", slog.String("error", err.Error()))
				os.Exit(2)
			}
			if entry.duration, err = time.ParseDuration(pcs[1]); err != nil {
				logger.Error("invalid duration", slog.String("error", err.Error()))
				os.Exit(2)
			}
			if entry.category, err = strconv.Atoi(pcs[2]); err != nil {
				logger.Error("invalid category", slog.String("error", err.Error()))
				os.Exit(2)
			}
			entry.notes = strings.TrimSpace(pcs[3])
			entries = append(entries, entry)
		}
		if scanner.Err() != nil {
			logger.Error("reading import file failed", slog.String("error", scanner.Err().Error()))
			os.Exit(1)
		}
		var tx *sql.Tx
		if tx, err = db.Begin(); err != nil {
			logger.Error("failed to start transaction", slog.String("error", err.Error()))
			os.Exit(1)
		}
		for _, entry := range entries {
			if _, err = myhours.ImportRecord(tx, entry.start, entry.duration, entry.category, entry.notes); err != nil {
				logger.Error("failed to import record", slog.String("error", err.Error()))
				os.Exit(1)
			}
		}
		if err = tx.Commit(); err != nil {
			logger.Error("failed to commit transaction", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("importing complete", slog.Int("numberOfEntries", len(entries)))
		os.Exit(0)
	}
	logger.Info("database initialized", slog.String("database", dbFile))
	// Run the application with given database
	if err = myhours.Run(db, myhours.UseLogger(logger)); err != nil {
		logger.Error("run error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
