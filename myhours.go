// Package myhours contains implementation for a bubbletea application for tracking
// time spent at work (or for personal projects).
//
// New returns the application model that is ready to be passed into a new
// bubbletea program.
package myhours

import (
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/help"
)

// Option defines a function that configures the application. Use with NewApplication
// or directly on MyHours.
type Option func(app *MyHours)

// UseLogger sets the logger for application. If nil, a logger based on
// slog.DiscardHandler is used as default.
func UseLogger(l *slog.Logger) Option {
	return func(app *MyHours) {
		if l == nil {
			l = slog.New(slog.DiscardHandler)
		}
		app.l = l
	}
}

// New returns an initialized MyHours model that can be passed into a
// bubbletea program for running the time tracking application.
//
// db must be some working implementation of Database. Reference implementations
// can be found under database sub-package.
//
// To use the returned model, call for example tea.NewProgram(model).Run()
func New(db Database, options ...Option) MyHours {
	h := help.New()
	h.Styles = styleHelp

	app := MyHours{
		db:    db,
		l:     slog.New(slog.DiscardHandler),
		help:  h,
		timer: newTimer(time.Millisecond * 250),
		state: state{
			reportPage: make([]int, 4),
		},
		keys: newKeymap(),
		viewNames: []string{
			"Timer",
			"Week",
			"Month",
			"Year",
		},
	}
	// apply options to customize the application.
	for _, opt := range options {
		opt(&app)
	}
	return app
}

type state struct {
	screenWidth      int
	screenHeight     int
	viewWidth        int
	viewHeight       int
	activeView       int
	timerCategoryID  int64
	activeRecordID   int64
	previousRecordID int64
	showHelp         bool
	ready            bool
	quitting         bool
	// reporting data fields
	reportLoading bool
	reportPage    []int
	reportTitle   string
	reportHeaders []string
	reportStyle   reportStyleFunc
	reportRows    [][]string
}

// MyHours is the my-hours application model. Keep track of the whole application
// state and implements tea.Model.
type MyHours struct {
	db         Database
	l          *slog.Logger
	settings   Settings
	categories []Category
	viewNames  []string
	keys       keymap
	state      state
	help       help.Model
	timer      timer
}

func incMax(v, max int) int {
	if v >= max {
		return max
	}
	return v + 1
}

func decMax(v, max int) int {
	if v > max {
		return max
	}
	return v - 1
}

func incWrap(v, min, max int) int {
	switch {
	case v >= max || v < min:
		return min
	default:
		return v + 1
	}
}

func decWrap(v, min, max int) int {
	switch {
	case v <= min || v > max:
		return max
	default:
		return v - 1
	}
}

func byIndex[T comparable](set []T, index int) T {
	if index <= 0 || index >= len(set) {
		return *new(T)
	}
	return set[index]
}

func findCategory(categories []Category, id int64) Category {
	for _, cat := range categories {
		if cat.ID == id {
			return cat
		}
	}
	return Category{ID: 0, Name: "unknown"}
}

func nextCategoryID(categories []Category, currentID int64) int64 {
	var idx int
	for idx = 0; idx < len(categories); idx++ {
		if categories[idx].ID == currentID {
			break
		}
	}
	if idx >= len(categories)-1 {
		idx = 0
	} else {
		idx++
	}
	return categories[idx].ID
}
