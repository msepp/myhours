package myhours

import (
	"log/slog"
	"strconv"

	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newYearlyReportView(db Database, l *slog.Logger) *reportView {
	return &reportView{
		id: nextViewID(),
		db: db,
		l:  l.With("view", "yearly"),
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(yearlyReportStyle).
			SetHeaders([]string{"Dates", "Month", "Clocked"}),
		name:         "Year",
		dateFilter:   yearlyFilter,
		rowFormatter: yearlyRows,
		tableHeader:  yearHeader,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}

func yearHeader(page int) string {
	from, _ := yearlyFilter(page)
	return strconv.FormatInt(int64(from.Year()), 10)
}
