package myhours

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func (app Application) renderNavigation() string {
	cat := activeCategory(app.categories, app.defaultCategory)
	var doc strings.Builder
	doc.WriteString(styleNavCap.Render("\uE0BA"))
	doc.WriteString(styleModeIndicator.Render("mode:"))
	doc.WriteString(styleModeIndicator.Foreground(cat.ForegroundColor()).Render(cat.name))
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

type keymap struct {
	switchGlobalCategory key.Binding
	switchTaskCategory   key.Binding
	tabNext              key.Binding
	tabPrev              key.Binding
	previousPage         key.Binding
	nextPage             key.Binding
	toggleTaskTimer      key.Binding
	openHelp             key.Binding
	closeHelp            key.Binding
	quit                 key.Binding
}

var appKeyMap = keymap{
	openHelp: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "help"),
	),
	closeHelp: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "close help"),
	),
	switchGlobalCategory: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "switch category"),
	),
	tabNext: key.NewBinding(
		key.WithKeys("right", "l", "n"),
		key.WithHelp("n", "next view"),
	),
	tabPrev: key.NewBinding(
		key.WithKeys("left", "h", "p"),
		key.WithHelp("h", "previous view"),
	),
	nextPage: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j", "forward in time"),
	),
	previousPage: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k", "back in time"),
	),
	toggleTaskTimer: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start/stop task"),
	),
	switchTaskCategory: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "switch task category"),
	),
	quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("q", "quit"),
	),
}
