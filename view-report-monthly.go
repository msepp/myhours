package myhours

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/msepp/myhours/report"
)

func newMonthlyReportView(db Database, l *slog.Logger) *reportView {
	return &reportView{
		id: nextViewID(),
		db: db,
		l:  l.With("view", "monthly"),
		report: report.New().
			SetTableBorder(lipgloss.NormalBorder()).
			SetStyleFunc(monthlyReportStyle).
			SetHeaders([]string{"Dates", "Week", "Clocked"}),
		name:         "Month",
		dateFilter:   monthlyFilter,
		rowFormatter: monthlyRows,
		tableHeader:  monthHeader,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}

func monthHeader(page int) string {
	from, until := monthlyFilter(page)
	return fmt.Sprintf("%s, %d (%s – %s)",
		from.Month().String(),
		from.Year(),
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}
