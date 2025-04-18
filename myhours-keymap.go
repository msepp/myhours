package myhours

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type keymap struct {
	switchGlobalCategory key.Binding
	switchTaskCategory   key.Binding
	nextTab              key.Binding
	prevTab              key.Binding
	prevReportPage       key.Binding
	nextReportPage       key.Binding
	toggleTaskTimer      key.Binding
	openHelp             key.Binding
	closeHelp            key.Binding
	quit                 key.Binding
}

func newKeymap() keymap {
	return keymap{
		openHelp: key.NewBinding(
			key.WithKeys(tea.KeyF1.String()),
			key.WithHelp("f1", "Help"),
			key.WithDisabled(),
		),
		closeHelp: key.NewBinding(
			key.WithKeys(tea.KeyF1.String()),
			key.WithHelp("f1", "Close help"),
			key.WithDisabled(),
		),
		switchGlobalCategory: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "Switch category"),
			key.WithDisabled(),
		),
		nextTab: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("l, →", "Next view"),
			key.WithDisabled(),
		),
		prevTab: key.NewBinding(
			key.WithKeys("left", "h", "p"),
			key.WithHelp("h, ←", "Previous view"),
			key.WithDisabled(),
		),
		nextReportPage: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j, ↓", "Forward in time"),
			key.WithDisabled(),
		),
		prevReportPage: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("k, ↑", "Back in time"),
			key.WithDisabled(),
		),
		toggleTaskTimer: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "Start/stop task"),
			key.WithDisabled(),
		),
		switchTaskCategory: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "Switch task category"),
			key.WithDisabled(),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "Quit"),
		),
	}
}
