package myhours

import (
	"fmt"
	"log/slog"
	"strconv"
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
			SetStyleFunc(monthReportStyle).
			SetHeaders([]string{"Dates", "Week", "Clocked"}),
		name:         "Month",
		dateFilter:   monthFilter,
		rowFormatter: monthRows,
		tableHeader:  monthHeader,
		categoryID:   2,
		keymap:       appKeyMap,
		page:         0,
		height:       0,
		width:        0,
	}
}

func monthHeader(page int) string {
	from, until := monthFilter(page)
	return fmt.Sprintf("%s, %d (%s – %s)",
		from.Month().String(),
		from.Year(),
		from.Format(time.DateOnly),
		until.Add(-1*time.Millisecond).Format(time.DateOnly),
	)
}

type monthlyReport struct {
	year      int
	month     time.Month
	firstDate string
	lastDate  string
	total     time.Duration
	weeks     []weeklyReport
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

func monthRows(records []Record) [][]string {
	var rows [][]string
	for _, m := range recordsAsMonths(records) {
		for _, w := range m.weeks {
			fd, ld := w.dateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				"W" + strconv.Itoa(w.weekNo),
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

func monthReportStyle(r, _ int, data []string) lipgloss.Style {
	if r < 0 || len(data) == 0 || data[0] == "" || data[0][0] != 'T' {
		return tableCellStyle
	}
	return tableSumRowStyle
}

func monthFilter(offset int) (time.Time, time.Time) {
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
