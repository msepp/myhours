package myhours

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/msepp/myhours/report"
)

var lastViewID int64

func nextViewID() int {
	return int(atomic.AddInt64(&lastViewID, 1))
}

type reportView struct {
	id           int
	db           Database
	l            *slog.Logger
	report       *report.Model
	name         string
	dateFilter   func(page int) (start time.Time, end time.Time)
	rowFormatter func(records []Record) [][]string
	tableHeader  func(page int) string
	categoryID   int64
	categories   []Category
	keymap       keymap
	page         int
	height       int
	width        int
}

func (view *reportView) Name() string { return view.name }

func (view *reportView) Update(message tea.Msg) tea.Cmd {
	switch msg := message.(type) {
	case viewAreaSizeMsg:
		view.width, view.height = msg.width, msg.height
	case updateCategoriesMsg:
		view.categories = msg.categories
		return view.updateData()
	case updateDefaultCategoryMsg:
		view.categoryID = msg.categoryID
		return view.updateData()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, view.keymap.previousPage):
			view.page--
			return view.updateData()
		case key.Matches(msg, view.keymap.nextPage):
			view.page = min(view.page+1, 0)
			return view.updateData()
		case key.Matches(msg, view.keymap.tabNext, view.keymap.tabPrev):
			return view.updateData()
		}
	}
	var cmd tea.Cmd
	view.report, cmd = view.report.Update(message)
	return cmd
}

func (view *reportView) updateData() tea.Cmd {
	from, before := view.dateFilter(view.page)
	records, err := view.db.RecordsInCategory(from, before, view.categoryID)
	if err != nil {
		view.l.Error("failed to fetch records", slog.String("error", err.Error()))
		return nil
	}
	return view.report.UpdateData(view.rowFormatter(records))
}

func (view *reportView) View() string {
	h, w := view.height, view.width
	table := view.report.SetSize(w, h).View()
	header := view.tableHeader(view.page)
	return header + "\n" + table
}

func (view *reportView) Init() tea.Cmd {
	var err error
	if view.categories, err = view.db.Categories(); err != nil {
		view.l.Error("failed to fetch categories", slog.String("error", err.Error()))
	}
	return nil
}

func (view *reportView) HelpKeys() []key.Binding {
	return []key.Binding{view.keymap.nextPage, view.keymap.previousPage}
}
