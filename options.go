package myhours

import "log/slog"

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
