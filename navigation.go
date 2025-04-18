package myhours

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (app Application) renderNavigation() string {
	cat := activeCategory(app.categories, app.defaultCategory)
	var doc strings.Builder
	doc.WriteString(styleNavCap.Render("\uE0BA"))
	doc.WriteString(styleModeIndicator.Render("mode:"))
	doc.WriteString(styleModeIndicator.Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString(styleNavInactive.Render("│"))
	var sections []string
	for i, view := range app.views {
		name := view.Name()
		var style lipgloss.Style
		_, _, isActive := i == 0, i == len(app.views)-1, i == app.activeView
		if isActive {
			name = "\uE617 " + name
			style = styleNavActive
		} else {
			style = styleNavInactive
		}
		sections = append(sections, style.Render(name))
	}
	doc.WriteString(strings.Join(sections, styleNavJoiner.Render("╱")))
	doc.WriteString(styleNavCap.Render("\uE0BC"))
	return doc.String()
}

type appKeys struct {
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

var appKeyMap = appKeys{
	openHelp: key.NewBinding(
		key.WithKeys(tea.KeyF1.String()),
		key.WithHelp("f1", "Help"),
	),
	closeHelp: key.NewBinding(
		key.WithKeys(tea.KeyF1.String()),
		key.WithHelp("f1", "Close help"),
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
	toggleTaskTimer: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "Start/stop task"),
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
