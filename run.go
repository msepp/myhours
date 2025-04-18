package myhours

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/help"
	tea "github.com/charmbracelet/bubbletea"
)

// Option defines a function that configures the application. Use with NewApplication
// or directly on Application.
type Option func(app *Application)

// UseLogger sets the logger for application. If nil, a logger based on
// slog.DiscardHandler is used as default.
func UseLogger(l *slog.Logger) Option {
	return func(app *Application) {
		if l == nil {
			l = slog.New(slog.DiscardHandler)
		}
		app.l = l
	}
}

// Run starts the myhours application using given database and optional options.
func Run(db Database, options ...Option) error {
	app := Application{
		db:     db,
		l:      slog.New(slog.DiscardHandler),
		keymap: appKeyMap,
		help:   help.New(),
	}
	app.help.Styles = styleHelp
	// apply options to customize the application.
	for _, opt := range options {
		opt(&app)
	}
	app.views = []viewRenderer{
		newTimerView(db, app.l, time.Millisecond*100),
		newWeeklyReportView(db, app.l),
		newMonthlyReportView(db, app.l),
		newYearlyReportView(db, app.l),
	}
	// fetch category options
	var err error
	if app.categories, err = db.Categories(); err != nil {
		return fmt.Errorf("load categories: %w", err)
	}
	app.defaultCategory = app.config.DefaultCategoryID
	appv1 := ApplicationV1{
		db: db,
		l:  app.l,
		models: models{
			help:  help.New(),
			timer: newTimerModel(time.Millisecond * 250),
		},
		state: appState{
			reportPage: make([]int, 4),
		},
		categories: app.categories,
		keys:       appKeyMap,
		viewNames: []string{
			"Timer",
			"Week",
			"Month",
			"Year",
		},
	}
	appv1.models.help.Styles = styleHelp
	// boot-up the bubbletea runtime with our application model.
	prog := tea.NewProgram(appv1, tea.WithAltScreen())
	if _, err = prog.Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
