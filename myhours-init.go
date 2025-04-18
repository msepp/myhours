package myhours

import (
	"fmt"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea"
)

// Init performs application initialization.
//
// Returns a set of commands that to update the application to a state where it
// is ready to function. Posts tea.Quit command if initiation fails at any point.
//
// Provides compatibility with tea.Model.
func (m MyHours) Init() tea.Cmd {
	commands := []tea.Cmd{
		func() tea.Msg {
			categories, err := m.db.Categories()
			if err != nil {
				err = fmt.Errorf("db.Categories: %w", err)
				m.l.Error("application init failed", slog.String("error", err.Error()))
				return tea.Quit()
			}
			return updateCategoriesMsg{categories: categories}
		},
		func() tea.Msg {
			settings, err := m.db.Settings()
			if err != nil {
				err = fmt.Errorf("db.Settings: %w", err)
				m.l.Error("application init failed", slog.String("error", err.Error()))
				return tea.Quit()
			}
			return updateSettingsMsg{settings: *settings}
		},
		func() tea.Msg {
			record, err := m.db.ActiveRecord()
			if err != nil {
				err = fmt.Errorf("db.ActiveRecord: %w", err)
				m.l.Error("application init failed", slog.String("error", err.Error()))
				return tea.Quit()
			}
			if record == nil {
				m.l.Info("no active record")
				return initTimerMsg{recordID: 0}
			}
			return initTimerMsg{recordID: record.ID, since: record.Start, category: record.CategoryID}
		},
	}
	return tea.Sequence(commands...)
}
