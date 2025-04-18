package myhours

import (
	"log/slog"

	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newWeeklyReportView(db Database, l *slog.Logger) *reportView {
	return &reportView{
		id: nextViewID(),
		db: db,
		l:  l.With("view", "weekly"),
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(weeklyReportStyle).
			SetHeaders([]string{"Date", "Day", "Clocked"}),
		name:         "Week",
		dateFilter:   weeklyFilter,
		rowFormatter: weeklyRows,
		tableHeader:  weeklyReportTitle,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}
