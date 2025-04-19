package myhours

import (
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// reportYearly defines a report for a yearly summary.
var reportYearly = report{
	headers: reportHeadersYearly,
	title:   reportTitleYearly,
	dates:   reportDatesYearly,
	styles:  reportStyleYearly,
	mapper:  reportRecordsYearly,
}

func reportHeadersYearly() []string {
	return []string{"Month", "Active days", "Duration"}
}

func reportTitleYearly(page int) string {
	from, _ := reportDatesYearly(page)
	return "Year " + strconv.FormatInt(int64(from.Year()), 10)
}

func reportDatesYearly(offset int) (time.Time, time.Time) {
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

func reportStyleYearly(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return styleTableCell
	}
	return styleTableSumRow
}

func reportRecordsYearly(records []Record) [][]string {
	var rows [][]string
	for _, y := range newYearlySummary(records) {
		activeDaysTotal := 0
		for _, m := range y.months {
			activeDays := m.activeDays()
			activeDaysTotal += activeDays
			rows = append(rows, []string{
				m.month.String(),
				strconv.Itoa(activeDays),
				m.total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{
			"Total",
			strconv.Itoa(activeDaysTotal),
			y.total.Truncate(time.Second).String(),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

type yearlySummary struct {
	year   int
	months []monthlySummary
	total  time.Duration
}

func newYearlySummary(records []Record) []yearlySummary {
	var (
		years []yearlySummary
		cy    *yearlySummary
	)
	for _, m := range newMonthlySummary(records) {
		if cy == nil || cy.year != m.year {
			years = append(years, yearlySummary{year: m.year})
			cy = &years[len(years)-1]
			for i := range 12 {
				cy.months = append(cy.months, monthlySummary{
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
