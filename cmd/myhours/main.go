package main

import (
	"bufio"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/msepp/myhours"
	"github.com/msepp/myhours/database/sqlite"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to get user config dir: %v\n", err)
		os.Exit(1)
	}
	var (
		logger     = slog.New(slog.DiscardHandler)
		doImport   bool
		verbose    bool
		silent     bool
		importFile = "import.txt"
		dbLocation = path.Join(configDir, "my-hours-cli")
		dbFile     = path.Join(dbLocation, "database.db")
	)
	flag.StringVar(&dbFile, "db", dbFile, "Database location")
	flag.BoolVar(&doImport, "import", doImport, "Run data import. -importFile selects import data location.")
	flag.StringVar(&importFile, "importFile", importFile, "File with import data. Must contain lines in format '2006-01-02T15:04:05.999999999Z07:00,<duration>,categoryInt,notes'. Notes can not contain newlines.")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&silent, "s", false, "Silence all log output")
	flag.Parse()

	if !silent {
		if verbose {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true}))
		} else {
			logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
		}
	}
	logger.Debug("opening database", slog.String("database", dbFile))
	// Initialize the database
	var dbConn *sql.DB
	if dbConn, err = sqlite.InitiateSQLiteDatabase(dbFile); err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	db := sqlite.NewSQLite(dbConn, sqlite.Logger(logger))
	// If user wants to do import, do it now that we know we have a destination
	// database ready.
	if doImport {
		logger.Debug("running import", slog.String("from", importFile), slog.String("to", dbFile))
		var f *os.File
		if f, err = os.Open(importFile); err != nil {
			logger.Error("reading import file failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		scanner := bufio.NewScanner(f)
		var entries []myhours.Record
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
			var entry myhours.Record
			if entry.Start, err = time.Parse(time.RFC3339Nano, pcs[0]); err != nil {
				logger.Error("invalid start time", slog.String("error", err.Error()))
				os.Exit(2)
			}
			var duration time.Duration
			if duration, err = time.ParseDuration(pcs[1]); err != nil {
				logger.Error("invalid duration", slog.String("error", err.Error()))
				os.Exit(2)
			}
			if entry.CategoryID, err = strconv.ParseInt(pcs[2], 10, 64); err != nil {
				logger.Error("invalid category", slog.String("error", err.Error()))
				os.Exit(2)
			}
			entry.Notes = strings.TrimSpace(pcs[3])
			entry.End = entry.Start.Add(duration)
			entries = append(entries, entry)
		}
		if scanner.Err() != nil {
			logger.Error("reading import file failed", slog.String("error", scanner.Err().Error()))
			os.Exit(1)
		}
		var result []int64
		if result, err = db.ImportRecords(entries); err != nil {
			logger.Error("failed to import records", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("importing complete", slog.Int("numberOfEntries", len(result)))
		os.Exit(0)
	}
	logger.Debug("database initialized", slog.String("database", dbFile))
	// Run the application with given database
	mh := myhours.New(db, myhours.UseLogger(logger))
	if _, err = tea.NewProgram(mh, tea.WithAltScreen()).Run(); err != nil {
		logger.Error("run error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Debug("Have a good day!")
}
