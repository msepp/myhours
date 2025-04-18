package myhours

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/lipgloss"
)

var (
	navColorBG         = lipgloss.AdaptiveColor{Light: "254", Dark: "16"}
	navColorFGActive   = lipgloss.AdaptiveColor{Light: "22", Dark: "40"}
	navColorFGInactive = lipgloss.AdaptiveColor{Light: "2", Dark: "243"}
	styleLoader        = lipgloss.NewStyle().Faint(true).Align(lipgloss.Center)
	styleNavCap        = lipgloss.NewStyle().Foreground(navColorBG)
	styleModeIndicator = lipgloss.NewStyle().Background(navColorBG).Foreground(navColorFGInactive).Padding(0, 0, 0, 1)
	styleNavJoiner     = lipgloss.NewStyle().Background(navColorBG).Foreground(navColorFGInactive)
	styleNavInactive   = lipgloss.NewStyle().Background(navColorBG).Foreground(navColorFGInactive).Padding(0, 1)
	styleNavActive     = lipgloss.NewStyle().Background(navColorBG).Foreground(navColorFGActive).Padding(0, 1)
	styleWindow        = lipgloss.NewStyle().Padding(0, 0, 0, 0).Margin(0, 0).Align(lipgloss.Center, lipgloss.Center)
	styleTimerLabel    = lipgloss.NewStyle().Bold(true).Width(10)
	styleTableCell     = lipgloss.NewStyle().Padding(0, 1)
	styleTableSumRow   = styleTableCell.Background(lipgloss.AdaptiveColor{Dark: "235", Light: "250"}).Foreground(lipgloss.AdaptiveColor{Dark: "195", Light: "20"})
	styleHelp          = help.Styles{
		Ellipsis:       lipgloss.NewStyle().Foreground(navColorFGInactive),
		ShortKey:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "4", Dark: "33"}),
		ShortDesc:      lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "238", Dark: "250"}),
		ShortSeparator: lipgloss.NewStyle().Foreground(navColorFGInactive),
		FullKey:        lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "4", Dark: "33"}),
		FullDesc:       lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "238", Dark: "250"}),
		FullSeparator:  lipgloss.NewStyle().Foreground(navColorFGInactive),
	}
	styleTimerContainer  = lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).Padding(1, 2)
	styleReportContainer = lipgloss.NewStyle().Padding(1, 0, 0, 0)
	styleReportTitle     = lipgloss.NewStyle().Margin(0, 2)
)
