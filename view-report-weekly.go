package myhours

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newWeeklyReportView() *weeklyReportView {
	return &weeklyReportView{
		page: 0,
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(weekReportStyle).
			SetHeaders([]string{"Date", "Day", "Clocked"}),
	}
}

type weeklyReportView struct {
	report *report.Model
	page   int
}

func (view *weeklyReportView) Name() string { return "Week" }

func (view *weeklyReportView) Update(app Application, message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.previousPage):
			view.page--
			return app, view.UpdateData(app)
		case key.Matches(msg, app.keymap.nextPage):
			view.page = min(view.page+1, 0)
			return app, view.UpdateData(app)
		case key.Matches(msg, app.keymap.tabNext, app.keymap.tabPrev, app.keymap.switchGlobalCategory):
			return app, view.UpdateData(app)
		}
	}
	var cmd tea.Cmd
	view.report, cmd = view.report.Update(message)
	return app, cmd
}

func (view *weeklyReportView) UpdateData(app Application) tea.Cmd {
	from, before := weekFilter(view.page)
	return view.report.UpdateData(weekRows(app.getRecords(from, before, &app.defaultCategory)))
}

func (view *weeklyReportView) View(_ Application, viewWidth, viewHeight int) string {
	if viewWidth > 80 {
		viewWidth = 80
	}
	table := view.report.SetSize(viewWidth, viewHeight).View()
	from, until := weekFilter(view.page)
	y, w := from.ISOWeek()
	header := fmt.Sprintf("Week %0d, %d (%s – %s)\n", w, y, from.Format(time.DateOnly), until.Add(-1*time.Millisecond).Format(time.DateOnly))
	return header + table
}

func (view *weeklyReportView) Init(_ Application) tea.Cmd {
	return nil
}

func (view *weeklyReportView) HelpKeys(keys keymap) []key.Binding {
	return []key.Binding{keys.nextPage, keys.previousPage}
}

func weekRows(records []dbRecord) [][]string {
	var rows [][]string
	for _, w := range recordsAsWeeks(records) {
		for _, d := range w.Days {
			rows = append(rows, []string{
				d.Date,
				d.WeekDay.String()[:3],
				d.Total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{"Total", "", w.Total.Truncate(time.Second).String()})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func weekReportStyle(row, _ int, _ []string) lipgloss.Style {
	if row == 7 {
		return tableSumRowStyle
	}
	return tableCellStyle
}

func weekFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now().AddDate(0, 0, offset*7) // week is always 7 days, so this still works.
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	wd--
	y, m, d := now.Date()
	base := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	switch wd {
	case 0:
		return base, base.AddDate(0, 0, 7)
	case 6:
		return base.AddDate(0, 0, -6), base.AddDate(0, 0, 1)
	default:
		return base.AddDate(0, 0, -1*wd), base.AddDate(0, 0, 7-wd)
	}
}
