package myhours

import (
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// reportMonthly defines a report for a monthly summary.
var reportMonthly = report{
	headers: reportHeadersMonthly,
	title:   reportTitleMonthly,
	dates:   reportDatesMonthly,
	styles:  reportStyleMonthly,
	mapper:  reportRecordsMonthly,
}

func reportHeadersMonthly() []string {
	return []string{"Week", "Dates", "Duration"}
}

func reportTitleMonthly(page int) string {
	from, until := reportDatesMonthly(page)
	return fmt.Sprintf("%s, %d (%s – %s)",
		from.Month().String(),
		from.Year(),
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

func reportDatesMonthly(offset int) (time.Time, time.Time) {
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

func reportStyleMonthly(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return styleTableCell
	}
	return styleTableSumRow
}

func reportRecordsMonthly(records []Record) [][]string {
	var rows [][]string
	for _, m := range newMonthlySummary(records) {
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

type monthlySummary struct {
	year      int
	month     time.Month
	firstDate string
	lastDate  string
	total     time.Duration
	weeks     []weeklySummary
}

func (s monthlySummary) activeDays() int {
	active := 0
	for _, w := range s.weeks {
		for _, d := range w.days {
			if d.total > 0 {
				active++
			}
		}
	}
	return active
}

func newMonthlySummary(records []Record) []monthlySummary {
	var (
		months []monthlySummary
		cm     *monthlySummary
		cw     *weeklySummary
	)
	for _, record := range records {
		start := record.Start.In(time.Local)
		y, m, _ := start.Date()
		if cm == nil || cm.month != m || cm.year != y {
			first := time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
			last := time.Date(y, m+1, -1, 0, 0, 0, 0, time.Local)
			months = append(months, monthlySummary{
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
					wr := weeklySummary{
						year:     y,
						weekNo:   weekNo,
						startsOn: first.Weekday(),
					}
					// seed the days of the week to get a full week.
					dd := first
					_, wNo := dd.ISOWeek()
					for wNo == weekNo && dd.Month() == m {
						wr.days = append(wr.days, dailySummary{
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
		_, weekNo := record.Start.In(time.Local).ISOWeek()
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
		d := record.Duration()
		cm.total += d
		cw.total += d
		date := start.Format(time.DateOnly)
		for i := range cw.days {
			if cw.days[i].date != date {
				continue
			}
			cw.days[i].total += d
			break
		}
		wd := start.Weekday()
		if wd == 0 {
			wd = 7
		}
		if cw.startsOn == time.Sunday {
			wd = -7
		} else {
			wd -= cw.startsOn
		}
		cw.days[wd].total += d
		if record.Notes != "" {
			cw.days[wd].notes = append(cw.days[wd].notes, record.Notes)
		}
	}
	return months
}
