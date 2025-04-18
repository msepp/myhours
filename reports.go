package myhours

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func weeklyReportStyle(row, _ int, _ []string) lipgloss.Style {
	if row == 7 {
		return styleTableSumRow
	}
	return styleTableCell
}

func monthlyReportStyle(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return styleTableCell
	}
	return styleTableSumRow
}

func yearlyReportStyle(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return styleTableCell
	}
	return styleTableSumRow
}

func weeklyReportTitle(page int) string {
	from, until := weeklyFilter(page)
	y, w := from.ISOWeek()
	return fmt.Sprintf("Week %0d, %d (%s – %s)",
		w,
		y,
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

func monthlyReportTitle(page int) string {
	from, until := monthlyFilter(page)
	return fmt.Sprintf("%s, %d (%s – %s)",
		from.Month().String(),
		from.Year(),
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

func yearlyReportTitle(page int) string {
	from, _ := yearlyFilter(page)
	return "Year " + strconv.FormatInt(int64(from.Year()), 10)
}

func weeklyReportHeaders() []string {
	return []string{"Weekday", "Date", "Duration"}
}

func monthlyReportHeaders() []string {
	return []string{"Week", "Dates", "Duration"}
}

func yearlyReportHeaders() []string {
	return []string{"Month", "Active days", "Duration"}
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

func yearlyRows(records []Record) [][]string {
	var rows [][]string
	for _, y := range recordsAsYears(records) {
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

func yearlyFilter(offset int) (time.Time, time.Time) {
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

type monthlyReport struct {
	year      int
	month     time.Month
	firstDate string
	lastDate  string
	total     time.Duration
	weeks     []weeklyReport
}

func (r monthlyReport) activeDays() int {
	active := 0
	for _, w := range r.weeks {
		for _, d := range w.days {
			if d.total > 0 {
				active++
			}
		}
	}
	return active
}

func (r monthlyReport) dateRange() (string, string) {
	if len(r.weeks) == 0 {
		return "", ""
	}
	if len(r.weeks) == 1 {
		return r.weeks[0].dateRange()
	}
	first, _ := r.weeks[0].dateRange()
	_, last := r.weeks[len(r.weeks)-1].dateRange()
	return first, last
}

func recordsAsMonths(records []Record) []monthlyReport {
	var (
		months []monthlyReport
		cm     *monthlyReport
		cw     *weeklyReport
	)
	for _, r := range records {
		start := r.Start.In(time.Local)
		y, m, _ := start.Date()
		if cm == nil || cm.month != m || cm.year != y {
			first := time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
			last := time.Date(y, m+1, -1, 0, 0, 0, 0, time.Local)
			months = append(months, monthlyReport{
				year:      y,
				month:     m,
				firstDate: first.Format(time.DateOnly),
				lastDate:  last.Format(time.DateOnly),
			})
			cm = &months[len(months)-1]
			// create the month weeks
			prevWeekNo := 0
			for !first.After(last) {
				if _, weekNo := first.ISOWeek(); prevWeekNo < weekNo {
					prevWeekNo = weekNo
					wr := weeklyReport{
						year:   y,
						weekNo: weekNo,
					}
					// seed the days of the week to get a full week.
					dd := first
					_, wNo := dd.ISOWeek()
					for wNo == weekNo && dd.Month() == m {
						wr.days = append(wr.days, dailyReport{
							date:    dd.Format(time.DateOnly),
							weekDay: dd.Weekday(),
							month:   dd.Month(),
						})
						dd = dd.AddDate(0, 0, 1)
						_, wNo = dd.ISOWeek()
					}
					cm.weeks = append(cm.weeks, wr)
				}
				first = first.AddDate(0, 0, 1)
			}
			cw = nil
		}
		_, weekNo := r.Start.In(time.Local).ISOWeek()
		if cw == nil || cw.weekNo != weekNo {
			for i, w := range cm.weeks {
				if w.weekNo == weekNo {
					cw = &cm.weeks[i]
				}
			}
			if cw == nil {
				panic("week not set somehow!")
			}
		}
		cm.total += r.Duration
		cw.total += r.Duration
		date := start.Format(time.DateOnly)
		for i := range cw.days {
			if cw.days[i].date != date {
				continue
			}
			cw.days[i].total += r.Duration
			break
		}
		wd := start.Weekday()
		if wd == 0 {
			wd = 7
		}
		wd--
		cw.days[wd].total += r.Duration
		if r.Notes != "" {
			cw.days[wd].notes = append(cw.days[wd].notes, r.Notes)
		}
	}
	return months
}

func monthlyRows(records []Record) [][]string {
	var rows [][]string
	for _, m := range recordsAsMonths(records) {
		for _, w := range m.weeks {
			fd, ld := w.dateRange()
			rows = append(rows, []string{
				"W" + strconv.Itoa(w.weekNo),
				fd + " – " + ld,
				w.total.Truncate(time.Second).String(),
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			m.total.Truncate(time.Second).String(),
		})
	}
	if len(rows) == 0 {
		rows = append(rows, []string{"NO DATA", "NO DATA", "NO DATA"})
	}
	return rows
}

func monthlyFilter(offset int) (time.Time, time.Time) {
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

func weeklyRows(records []Record) [][]string {
	var rows [][]string
	for _, w := range recordsAsWeeks(records) {
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

func weeklyFilter(offset int) (time.Time, time.Time) {
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
