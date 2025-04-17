package myhours

import (
	"log/slog"
	"strconv"
	"time"

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
			SetStyleFunc(yearReportStyle).
			SetHeaders([]string{"Dates", "Month", "Clocked"}),
		name:         "Year",
		dateFilter:   yearFilter,
		rowFormatter: yearRows,
		tableHeader:  yearHeader,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}

func yearHeader(page int) string {
	from, _ := yearFilter(page)
	return strconv.FormatInt(int64(from.Year()), 10)
}

type yearlyReport struct {
	year   int
	months []monthlyReport
	total  time.Duration
}

func (r yearlyReport) dateRange() (string, string) {
	if len(r.months) == 0 {
		return "", ""
	}
	if len(r.months) == 1 {
		return r.months[0].dateRange()
	}
	first, _ := r.months[0].dateRange()
	_, last := r.months[len(r.months)-1].dateRange()
	return first, last
}

func recordsAsYears(records []Record) []yearlyReport {
	var (
		years []yearlyReport
		cy    *yearlyReport
	)
	for _, m := range recordsAsMonths(records) {
		if cy == nil || cy.year != m.year {
			years = append(years, yearlyReport{year: m.year})
			cy = &years[len(years)-1]
			for i := range 12 {
				cy.months = append(cy.months, monthlyReport{
					year:  m.year,
					month: time.Month(i + 1),
				})
			}
		}
		cy.months[m.month-1] = m
		cy.total += m.total
	}
	return years
}

func yearRows(records []Record) [][]string {
	var rows [][]string
	for _, y := range recordsAsYears(records) {
		for _, m := range y.months {
			fd, ld := m.dateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				m.month.String(),
				m.total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			y.total.Truncate(time.Second).String(),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func yearReportStyle(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return tableCellStyle
	}
	return tableSumRowStyle
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
