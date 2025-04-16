package myhours

import "github.com/charmbracelet/lipgloss"

var (
	inactiveTabBorder = tabBorderWithBottom(" ")
	activeTabBorder   = tabBorderWithBottom("─")
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 0)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(0, 0).Margin(0, 1).Align(lipgloss.Center, lipgloss.Center).Border(lipgloss.NormalBorder())
)

func tabBorderWithBottom(middle string) lipgloss.Border {
	border := lipgloss.Border{}
	border.Bottom = middle
	return border
}

func (app Application) renderNavigation() string {
	var renderedTabs []string
	for i, view := range app.views {
		var style lipgloss.Style
		_, _, isActive := i == 0, i == len(app.views)-1, i == app.activeView
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(view.Name()))
	}
	return lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)
}
