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

func newTimerView(db Database, l *slog.Logger, interval time.Duration) *timerView {
	return &timerView{
		id:     nextViewID(),
		timer:  stopwatch.NewWithInterval(interval),
		db:     db,
		l:      l.With("view", "timer"),
		keymap: appKeyMap,
	}
}

type timerView struct {
	id             int
	db             Database
	l              *slog.Logger
	timer          stopwatch.Model
	keymap         appKeys
	categories     []Category
	width          int
	activeCategory int64
	activeRecordID int64
	prevRecordID   int64
}

func (view *timerView) Name() string { return "Task" }

func (view *timerView) Update(message tea.Msg) tea.Cmd {
	switch msg := message.(type) {
	case viewAreaSizeMsg:
		view.width = msg.width
	case reHydrateMsg:
		view.activeCategory = msg.category
		view.activeRecordID = msg.recordID
		return view.timer.StartFrom(msg.since)
	case updateCategoriesMsg:
		view.categories = msg.categories
	case updateDefaultCategoryMsg:
		// If active category is less than 1, set it to the global value.
		if view.activeCategory < 1 {
			view.activeCategory = msg.categoryID
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, view.keymap.switchTaskCategory):
			cat := nextCategory(view.categories, view.activeCategory)
			view.activeCategory = cat.ID
			// if there's an active task running, switch the category for it.
			if view.activeRecordID > 0 {
				if err := view.db.UpdateRecordCategory(view.activeRecordID, view.activeCategory); err != nil {
					view.l.Error("failed to update active record category", slog.String("error", err.Error()))
				}
			}
			return nil
		case key.Matches(msg, view.keymap.toggleTaskTimer):
			switch view.timer.Running() {
			case false:
				start := time.Now()
				var err error
				if view.activeRecordID, err = view.db.StartRecord(start, view.activeCategory, ""); err != nil {
					view.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				return view.timer.StartFrom(start)
			case true:
				now := time.Now()
				if err := view.db.FinishRecord(view.activeRecordID, view.timer.Since(), now, ""); err != nil {
					view.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				view.prevRecordID = view.activeRecordID
				view.activeRecordID = 0
				return view.timer.Stop()
			}
		}
	}
	var cmd tea.Cmd
	view.timer, cmd = view.timer.Update(message)
	return cmd
}

func (view *timerView) View() string {
	var doc strings.Builder
	cat := activeCategory(view.categories, view.activeCategory)
	w := min(40, view.width)
	elapsed := view.timer.View()
	started := view.timer.Since()
	style := lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(cat.ForegroundColor()).Padding(1, 2).Width(w)
	doc.WriteString(styleTimerLabel.Render("Tracking:"))
	doc.WriteString(lipgloss.NewStyle().Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Elapsed:"))
	doc.WriteString(elapsed)
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Started:"))
	if !started.IsZero() {
		doc.WriteString(started.Format(time.DateTime))
	}
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Task ID:"))
	switch {
	case view.activeRecordID > 0:
		doc.WriteString(strconv.FormatInt(view.activeRecordID, 10))
	case view.prevRecordID > 0:
		doc.WriteString(strconv.FormatInt(view.prevRecordID, 10))
	}
	return style.Render(doc.String())
}

func (view *timerView) Init() tea.Cmd {
	view.width = 80
	return nil
}

func (view *timerView) HelpKeys() []key.Binding {
	return []key.Binding{view.keymap.switchTaskCategory, view.keymap.toggleTaskTimer}
}
