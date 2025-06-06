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
	// disable all keys by default (except quite). They'll be enabled once app
	// is ready.
	app.keys.switchGlobalCategory.SetEnabled(false)
	app.keys.switchTaskCategory.SetEnabled(false)
	app.keys.nextTab.SetEnabled(false)
	app.keys.prevTab.SetEnabled(false)
	app.keys.prevReportPage.SetEnabled(false)
	app.keys.nextReportPage.SetEnabled(false)
	app.keys.startRecord.SetEnabled(false)
	app.keys.stopRecord.SetEnabled(false)
	app.keys.newRecord.SetEnabled(false)
	app.keys.openHelp.SetEnabled(false)
	app.keys.closeHelp.SetEnabled(false)
	// apply options to customize the application.
	for _, opt := range options {
		opt(&app)
	}
	return app
}

type state struct {
	altScreen    bool
	screenWidth  int
	screenHeight int
	viewWidth    int
	viewHeight   int
	activeView   int
	activeRecord Record
	showHelp     bool
	ready        bool
	quitting     bool
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

// indexOrZero returns the value from set in given index, or if index does not
// exist in the given slice, the zero value of type T.
func indexOrZero[T comparable](set []T, index int) T {
	if index <= 0 || index >= len(set) {
		return *new(T)
	}
	return set[index]
}

// findCategory from given slice by id. Returns placeholder value with ID zero
// if no Category was found with given id.
func findCategory(categories []Category, id int64) Category {
	for _, cat := range categories {
		if cat.ID == id {
			return cat
		}
	}
	return Category{ID: 0, Name: "unknown"}
}

// nextCategoryID returns ID of the next Category from given slice, using the
// currentID as the starting point. If current ID is the last entry in given slice,
// returns the ID of the first Category.
//
// If categories is empty, returns the given currentID. If currentID can not be
// found in the slice, returns ID of first Category.
func nextCategoryID(categories []Category, currentID int64) int64 {
	// if there's no categories, just return current ID, what ever that means
	if len(categories) == 0 {
		return currentID
	}
	// also, if there's just one category, we'll return that.
	if len(categories) == 1 {
		return categories[0].ID
	}
	// find the category that matches current one, get next index in list of
	// categories and return that, wrapping around if necessary.
	for i, cat := range categories {
		if cat.ID != currentID {
			continue
		}
		nextIdx := incWrap(i, 0, len(categories)-1)
		return categories[nextIdx].ID
	}
	// given category doesn't exist in current set, odd. Return the first one we
	// have.
	return categories[0].ID
}
