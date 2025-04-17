package myhours

import (
	"fmt"
	"log/slog"
	"time"

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
			SetStyleFunc(weekReportStyle).
			SetHeaders([]string{"Date", "Day", "Clocked"}),
		name:         "Week",
		dateFilter:   weekFilter,
		rowFormatter: weekRows,
		tableHeader:  weekHeader,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}

func weekHeader(page int) string {
	from, until := weekFilter(page)
	y, w := from.ISOWeek()
	return fmt.Sprintf("Week %0d, %d (%s – %s)",
		w,
		y,
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

type dailyReport struct {
	date    string
	weekDay time.Weekday
	month   time.Month
	total   time.Duration
	notes   []string
}

type weeklyReport struct {
	year   int
	weekNo int
	total  time.Duration
	days   []dailyReport
}

func (r weeklyReport) dateRange() (string, string) {
	if len(r.days) == 0 {
		return "", ""
	}
	if len(r.days) == 1 {
		return r.days[0].date, r.days[0].date
	}
	return r.days[0].date, r.days[len(r.days)-1].date
}

func recordsAsWeeks(records []Record) []weeklyReport {
	var (
		weeks []weeklyReport
		cw    *weeklyReport
	)
	for _, record := range records {
		// show dates in local time. They are stored in UTC.
		start := record.Start.In(time.Local)
		y, weekNo := start.In(time.Local).ISOWeek()
		wd := int(start.Weekday())
		if wd == 0 { // Sunday is zero. Horrible.
			wd = 7
		}
		wd--
		if cw == nil || cw.year != y || cw.weekNo != weekNo {
			weeks = append(weeks, weeklyReport{year: y, weekNo: weekNo})
			cw = &weeks[len(weeks)-1]
			// seed the days of the week to get a full week.
			for i := range 7 {
				d := start.AddDate(0, 0, -1*(wd-i))
				cw.days = append(cw.days, dailyReport{
					date:    d.Format(time.DateOnly),
					weekDay: d.Weekday(),
					month:   d.Month(),
				})
			}
		}
		cw.total += record.Duration
		cd := &cw.days[wd]
		cd.total += record.Duration
		if record.Notes != "" {
			cd.notes = append(cd.notes, record.Notes)
		}
	}
	return weeks
}

func weekRows(records []Record) [][]string {
	var rows [][]string
	for _, w := range recordsAsWeeks(records) {
		for _, d := range w.days {
			rows = append(rows, []string{
				d.date,
				d.weekDay.String()[:3],
				d.total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{"Total", "", w.total.Truncate(time.Second).String()})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func weekReportStyle(row, _ int, _ []string) lipgloss.Style {
	if row == 7 {
		return styleTableSumRow
	}
	return styleTableCell
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
