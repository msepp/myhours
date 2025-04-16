package myhours

import (
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newYearlyReportView() *yearlyReportView {
	return &yearlyReportView{
		page: 0,
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(yearReportStyle).
			SetHeaders([]string{"Dates", "Month", "Clocked"}),
	}
}

type yearlyReportView struct {
	report *report.Model
	page   int
}

func (view *yearlyReportView) Name() string { return "Year" }

func (view *yearlyReportView) Update(app Application, message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.previousPage):
			view.page--
			return app, view.report.UpdateData(yearRows(app.getRecords(yearFilter(view.page))))
		case key.Matches(msg, app.keymap.nextPage):
			view.page = min(view.page+1, 0)
			return app, view.report.UpdateData(yearRows(app.getRecords(yearFilter(view.page))))
		case key.Matches(msg, app.keymap.tabNext, app.keymap.tabNext):
			return app, view.report.UpdateData(yearRows(app.getRecords(yearFilter(view.page))))
		}
	}
	var cmd tea.Cmd
	view.report, cmd = view.report.Update(message)
	return app, cmd
}

func (view *yearlyReportView) View(_ Application, viewWidth, viewHeight int) string {
	if viewWidth > 80 {
		viewWidth = 80
	}
	table := view.report.SetSize(viewWidth, viewHeight).View()
	from, _ := yearFilter(view.page)
	header := strconv.FormatInt(int64(from.Year()), 10)
	return header + "\n" + table
}

func (view *yearlyReportView) Init(_ Application) tea.Cmd {
	return nil
}

func (view *yearlyReportView) ShortHelpKeys(keys keymap) []key.Binding {
	return []key.Binding{keys.nextPage, keys.previousPage}
}

func yearRows(records []dbRecord) [][]string {
	var rows [][]string
	for _, y := range recordsAsYears(records) {
		for _, m := range y.Months {
			fd, ld := m.DateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				m.Month.String(),
				m.Total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			y.Total.Truncate(time.Second).String(),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func yearReportStyle(r, _ int, data []string) lipgloss.Style {
	s := lipgloss.NewStyle().Padding(0, 1)
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return s
	}
	return s.Background(lipgloss.AdaptiveColor{Dark: "#FFF", Light: "#000"}).Foreground(lipgloss.AdaptiveColor{Dark: "#000", Light: "#FFF"})
}

func yearFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now()
	y, _, _ := now.Date()
	// first day of current year, minus as many years as offset says
	from := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local).AddDate(offset, 0, 0)
	before := from.AddDate(1, 0, 0)
	return from, before
}
