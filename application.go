package myhours

import (
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/msepp/myhours/stopwatch"
)

// Application is the myhours application handle / model. Implements the application
// logic for time tracking.
type Application struct {
	l         *slog.Logger
	db        *sql.DB
	stopwatch stopwatch.Model
	keymap    keymap
	help      help.Model
	quitting  bool
}

type keymap struct {
	start key.Binding
	stop  key.Binding
	reset key.Binding
	quit  key.Binding
}

func (m Application) Init() tea.Cmd {
	m.stopwatch.Init()
	return nil
}

func (m Application) View() string {
	// Note: you could further customize the time output by getting the
	// duration from m.stopwatch.Elapsed(), which returns a time.Duration, and
	// skip m.stopwatch.View() altogether.
	s := m.stopwatch.View() + "\n"
	if !m.quitting {
		s = "Elapsed: " + s
		s += m.helpView()
	}
	return s
}

func (m Application) helpView() string {
	return "\n" + m.help.ShortHelpView([]key.Binding{
		m.keymap.start,
		m.keymap.stop,
		m.keymap.reset,
		m.keymap.quit,
	})
}

func (m Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keymap.quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, m.keymap.reset):
			return m, m.stopwatch.Reset()
		case key.Matches(msg, m.keymap.start, m.keymap.stop):
			if m.stopwatch.Running() {
				m.stopwatch.Stop()
				t0 := m.stopwatch.Since()
				t1 := t0.Add(m.stopwatch.Elapsed())
				if err := insertRecord(m.db, t0, t1, 2, "temporary notes"); err != nil {
					m.l.Error("failed to store record", slog.String("error", err.Error()))
				}
			}
			m.keymap.stop.SetEnabled(!m.stopwatch.Running())
			m.keymap.start.SetEnabled(m.stopwatch.Running())
			return m, m.stopwatch.Toggle()
		}
	}
	var cmd tea.Cmd
	m.stopwatch, cmd = m.stopwatch.Update(msg)
	return m, cmd
}

// Run starts the myhours application using given database and optional options.
func Run(db *sql.DB, options ...Option) error {
	// Setup the application components and key-bindings
	appModel := Application{
		db:        db,
		l:         slog.New(slog.DiscardHandler),
		stopwatch: stopwatch.NewWithInterval(time.Millisecond * 100),
		keymap: keymap{
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
			),
			quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("q", "quit"),
			),
		},
		help: help.New(),
	}
	// apply options to customize the application.
	for _, opt := range options {
		opt(&appModel)
	}
	// boot-up the bubbletea runtime with our application model.
	if _, err := tea.NewProgram(appModel).Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
