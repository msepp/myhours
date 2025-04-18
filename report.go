package myhours

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

type reportStyleFunc func(row, col int, rowData []string) lipgloss.Style
type reportMapperFunc func([]Record) [][]string
type reportDatesFunc func(int) (time.Time, time.Time)
type reportTitleFunc func(int) string
type reportHeaderFunc func() []string

// report is a common spec for reports, defining the minimum requirements.
type report struct {
	headers reportHeaderFunc
	mapper  reportMapperFunc
	dates   reportDatesFunc
	title   reportTitleFunc
	styles  reportStyleFunc
}
