package myhours

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newMonthlyReportView() *monthlyReportView {
	return &monthlyReportView{
		page: 0,
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(monthReportStyle).
			SetHeaders([]string{"Dates", "Week", "Clocked"}),
	}
}

type monthlyReportView struct {
	report *report.Model
	page   int
}

func (view *monthlyReportView) Name() string { return "Month" }

func (view *monthlyReportView) Update(app Application, message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.previousPage):
			view.page--
			return app, view.report.UpdateData(monthRows(app.getRecords(monthFilter(view.page))))
		case key.Matches(msg, app.keymap.nextPage):
			view.page = min(view.page+1, 0)
			return app, view.report.UpdateData(monthRows(app.getRecords(monthFilter(view.page))))
		case key.Matches(msg, app.keymap.tabNext, app.keymap.tabPrev):
			return app, view.report.UpdateData(monthRows(app.getRecords(monthFilter(view.page))))
		}
	}
	var cmd tea.Cmd
	view.report, cmd = view.report.Update(message)
	return app, cmd
}

func (view *monthlyReportView) View(_ Application, viewWidth, viewHeight int) string {
	if viewWidth > 80 {
		viewWidth = 80
	}
	table := view.report.SetSize(viewWidth, viewHeight).View()
	from, until := monthFilter(view.page)
	header := fmt.Sprintf("%s, %d (%s – %s)\n", from.Month().String(), from.Year(), from.Format(time.DateOnly), until.Add(-1*time.Millisecond).Format(time.DateOnly))
	return header + table
}

func (view *monthlyReportView) Init(_ Application) tea.Cmd {
	return nil
}

func (view *monthlyReportView) ShortHelpKeys(keys keymap) []key.Binding {
	return []key.Binding{keys.nextPage, keys.previousPage}
}

func monthRows(records []dbRecord) [][]string {
	var rows [][]string
	for _, m := range recordsAsMonths(records) {
		for _, w := range m.Weeks {
			fd, ld := w.DateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				"W" + strconv.Itoa(w.WeekNo),
				w.Total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			m.Total.Truncate(time.Second).String(),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func monthReportStyle(r, _ int, data []string) lipgloss.Style {
	s := lipgloss.NewStyle().Padding(0, 1)
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return s
	}
	return s.Background(lipgloss.AdaptiveColor{Dark: "#FFF", Light: "#000"}).Foreground(lipgloss.AdaptiveColor{Dark: "#000", Light: "#FFF"})
}

func monthFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now()
	y, m, _ := now.Date()
	// first day of current month, minus as many months as offset says
	from := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
	before := time.Date(y, m+1, 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
	return from, before
}
