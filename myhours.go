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

func (m MyHours) reportPageNo() int {
	if m.state.activeView >= len(m.state.reportPage) {
		return 0
	}
	return m.state.reportPage[m.state.activeView]
}

func (m MyHours) category(id int64) Category {
	for _, cat := range m.categories {
		if cat.ID == id {
			return cat
		}
	}
	return Category{ID: 0, Name: "unknown"}
}

func (m MyHours) nextCategoryID(categoryID int64) int64 {
	var idx int
	for idx = 0; idx < len(m.categories); idx++ {
		if m.categories[idx].ID == categoryID {
			break
		}
	}
	if idx >= len(m.categories)-1 {
		idx = 0
	} else {
		idx++
	}
	return m.categories[idx].ID
}
