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
	app.help.Styles = helpStyle
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
	// load configuration
	var settings *Settings
	if settings, err = db.Settings(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	app.config = *settings
	app.defaultCategory = app.config.DefaultCategory
	// boot-up the bubbletea runtime with our application model.
	prog := tea.NewProgram(app, tea.WithAltScreen())
	if _, err = prog.Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
