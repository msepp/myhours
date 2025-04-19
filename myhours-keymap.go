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
	startRecord          key.Binding
	stopRecord           key.Binding
	newRecord            key.Binding
	openHelp             key.Binding
	closeHelp            key.Binding
	quit                 key.Binding
}

func newKeymap() keymap {
	return keymap{
		openHelp: key.NewBinding(
			key.WithKeys(tea.KeyF1.String()),
			key.WithHelp("f1", "Help"),
		),
		closeHelp: key.NewBinding(
			key.WithKeys(tea.KeyEsc.String(), tea.KeyF1.String()),
			key.WithHelp("esc", "Close help"),
		),
		switchGlobalCategory: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "Switch category"),
		),
		nextTab: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("l, →", "Next view"),
		),
		prevTab: key.NewBinding(
			key.WithKeys("left", "h", "p"),
			key.WithHelp("h, ←", "Previous view"),
		),
		nextReportPage: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("j, ↓", "Forward in time"),
		),
		prevReportPage: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("k, ↑", "Back in time"),
		),
		startRecord: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "Start"),
		),
		stopRecord: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "Stop"),
		),
		newRecord: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "New"),
		),
		switchTaskCategory: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "Switch task category"),
		),
		quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("q", "Quit"),
		),
	}
}
