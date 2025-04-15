package main

import (
	_ "embed"
	"flag"
	"log/slog"
	"os"
	"path"

	"github.com/msepp/myhours"

	_ "github.com/glebarez/go-sqlite"
)

var logger = slog.New(slog.NewTextHandler(os.Stderr, nil))

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		logger.Warn("failed to determine user config directory", slog.String("error", err.Error()))
	}
	var (
		dbLocation = path.Join(configDir, "my-hours-cli")
		dbFile     = path.Join(dbLocation, "database.db")
	)
	flag.StringVar(&dbFile, "db", dbFile, "Database location")
	flag.Parse()
	logger.Info("opening database", slog.String("database", dbFile))
	// Initialize the database
	db, err := myhours.NewSQLiteDatabase(dbFile)
	if err != nil {
		logger.Error("failed to open database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("database initialized", slog.String("database", dbFile))
	// Run the application with given database
	if err = myhours.Run(db, myhours.UseLogger(logger)); err != nil {
		logger.Error("run error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
