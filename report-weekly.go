package myhours

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// reportWeekly defines a report for a weekly summary.
var reportWeekly = report{
	headers: reportHeadersWeekly,
	title:   reportTitleWeekly,
	dates:   reportDatesWeekly,
	styles:  reportStyleWeekly,
	mapper:  reportRecordsWeekly,
}

func reportHeadersWeekly() []string {
	return []string{"Weekday", "Date", "Duration"}
}

func reportTitleWeekly(page int) string {
	from, until := reportDatesWeekly(page)
	y, w := from.ISOWeek()
	return fmt.Sprintf("Week %0d, %d (%s – %s)",
		w,
		y,
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

func reportDatesWeekly(offset int) (time.Time, time.Time) {
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

func reportStyleWeekly(row, _ int, _ []string) lipgloss.Style {
	if row == 7 {
		return styleTableSumRow
	}
	return styleTableCell
}

func reportRecordsWeekly(records []Record) [][]string {
	var rows [][]string
	for _, w := range newWeeklySummary(records) {
		for _, d := range w.days {
			rows = append(rows, []string{
				d.weekDay.String()[:3],
				d.date,
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

type dailySummary struct {
	date    string
	weekDay time.Weekday
	month   time.Month
	total   time.Duration
	notes   []string
}

type weeklySummary struct {
	year   int
	weekNo int
	total  time.Duration
	days   []dailySummary
}

func (s weeklySummary) dateRange() (string, string) {
	if len(s.days) == 0 {
		return "", ""
	}
	if len(s.days) == 1 {
		return s.days[0].date, s.days[0].date
	}
	return s.days[0].date, s.days[len(s.days)-1].date
}

func newWeeklySummary(records []Record) []weeklySummary {
	var (
		weeks []weeklySummary
		cw    *weeklySummary
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
			weeks = append(weeks, weeklySummary{year: y, weekNo: weekNo})
			cw = &weeks[len(weeks)-1]
			// seed the days of the week to get a full week.
			for i := range 7 {
				d := start.AddDate(0, 0, -1*(wd-i))
				cw.days = append(cw.days, dailySummary{
					date:    d.Format(time.DateOnly),
					weekDay: d.Weekday(),
					month:   d.Month(),
				})
			}
		}
		d := record.Duration()
		cw.total += d
		cd := &cw.days[wd]
		cd.total += d
		if record.Notes != "" {
			cd.notes = append(cd.notes, record.Notes)
		}
	}
	return weeks
}
