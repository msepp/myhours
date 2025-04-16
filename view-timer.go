package myhours

import (
	"log/slog"
	"strconv"
	"strings"
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
	timer          stopwatch.Model
	activeCategory int64
	activeRecordID int64
	prevRecordID   int64
}

func (view *timerView) Name() string { return "Task" }

func (view *timerView) Update(app Application, message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case reHydrateMsg:
		view.activeCategory = msg.category
		view.activeRecordID = msg.recordID
		return app, view.timer.StartFrom(msg.since)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.switchTaskCategory):
			cat := nextCategory(app.categories, view.activeCategory)
			view.activeCategory = cat.id
			// if there's an active task running, switch the category for it.
			if view.activeRecordID > 0 {
				if err := app.setRecordCategory(view.activeRecordID, view.activeCategory); err != nil {
					app.l.Error("failed to update active record category", slog.String("error", err.Error()))
				}
			}
			return app, nil
		case key.Matches(msg, app.keymap.toggleTaskTimer):
			switch view.timer.Running() {
			case false:
				start := time.Now()
				var err error
				if view.activeRecordID, err = app.startRecord(start, view.activeCategory, ""); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				return app, view.timer.StartFrom(start)
			case true:
				now := time.Now()
				if err := app.finishRecord(view.activeRecordID, view.timer.Since(), now, ""); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				view.prevRecordID = view.activeRecordID
				view.activeRecordID = 0
				return app, view.timer.Stop()
			}
		}
	}
	var cmd tea.Cmd
	view.timer, cmd = view.timer.Update(message)
	return app, cmd
}

func (view *timerView) View(app Application, width, _ int) string {
	var doc strings.Builder
	cat := activeCategory(app.categories, view.activeCategory)
	if width > 40 {
		width = 40
	}
	elapsed := view.timer.View()
	started := view.timer.Since()
	style := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(cat.ForegroundColor()).Padding(1, 2).Width(width)
	doc.WriteString(timerLabel.Render("Tracking:"))
	doc.WriteString(lipgloss.NewStyle().Foreground(cat.ForegroundColor()).Render(cat.name))
	doc.WriteString("\n")
	doc.WriteString(timerLabel.Render("Elapsed:"))
	doc.WriteString(elapsed)
	doc.WriteString("\n")
	doc.WriteString(timerLabel.Render("Started:"))
	if !started.IsZero() {
		doc.WriteString(started.Format(time.DateTime))
	}
	doc.WriteString("\n")
	doc.WriteString(timerLabel.Render("Task ID:"))
	switch {
	case view.activeRecordID > 0:
		doc.WriteString(strconv.FormatInt(view.activeRecordID, 10))
	case view.prevRecordID > 0:
		doc.WriteString(strconv.FormatInt(view.prevRecordID, 10))
	}
	return style.Render(doc.String())
}

func (view *timerView) Init(app Application) tea.Cmd {
	view.activeCategory = app.defaultCategory
	return nil
}

func (view *timerView) HelpKeys(keys keymap) []key.Binding {
	return []key.Binding{keys.switchTaskCategory, keys.toggleTaskTimer}
}
