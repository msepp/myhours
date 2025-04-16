package myhours

import (
	"log/slog"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/stopwatch"
)

func newTimerView(interval time.Duration) *timerView {
	return &timerView{timer: stopwatch.NewWithInterval(interval)}
}

type timerView struct {
	timer stopwatch.Model
}

func (view *timerView) Name() string { return "Timer" }

func (view *timerView) Update(app Application, message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case reHydrateMsg:
		return app, view.timer.StartFrom(msg.since)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.start, app.keymap.stop):
			app.keymap.stop.SetEnabled(!view.timer.Running())
			app.keymap.start.SetEnabled(view.timer.Running())
			switch view.timer.Running() {
			case false:
				start := time.Now()
				var err error
				if app.activeRecordID, err = app.startRecord(start, 2, ""); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				return app, view.timer.StartFrom(start)
			case true:
				now := time.Now()
				if err := app.finishRecord(app.activeRecordID, view.timer.Since(), now, "fake notes"); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				app.activeRecordID = 0
				return app, view.timer.Reset(false)
			}
		}
	}
	var cmd tea.Cmd
	view.timer, cmd = view.timer.Update(message)
	return app, cmd
}

func (view *timerView) View(_ Application, _, _ int) string {
	return lipgloss.NewStyle().Bold(true).Render(view.timer.View())
}

func (view *timerView) Init(_ Application) tea.Cmd {
	return nil
}

func (view *timerView) ShortHelpKeys(keys keymap) []key.Binding {
	return []key.Binding{keys.start, keys.stop}
}
