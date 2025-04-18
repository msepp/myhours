package myhours

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// View renders the current model state.
//
// Provides compatibility with tea.Model.
func (m MyHours) View() string {
	if !m.state.ready {
		// still in init.
		return styleWindow.
			Height(m.state.screenHeight).
			Width(m.state.screenWidth).
			Align(lipgloss.Center).
			Render("Loading ...")
	}
	switch {
	case m.state.showHelp:
		return m.renderHelp()
	default:
		var view viewFunc
		switch m.state.activeView {
		case 0:
			view = m.renderTimer
		case 1, 2, 3:
			view = m.renderReport
		default:
			view = func(int, int) string { return "you should not get here.." }
		}
		return m.renderView(view)
	}
}

func (m MyHours) renderHelp() string {
	h := m.help
	h.Width = m.state.viewWidth
	return lipgloss.Place(
		m.state.viewWidth,
		m.state.viewHeight,
		lipgloss.Center,
		lipgloss.Center,
		// The layout here is really hacky with the pseudo keys to create segment
		// titles, but I'm so lazy I can't be bothered to do it right when this
		// works just fine.
		h.FullHelpView([][]key.Binding{
			// global keys
			{
				key.NewBinding(key.WithHelp("", "Global:"), key.WithKeys("")),
				m.keys.switchGlobalCategory,
				m.keys.nextTab,
				m.keys.prevTab,
				m.keys.quit,
				m.keys.closeHelp,
			},
			// view specific keys
			{
				// timer view keys
				key.NewBinding(key.WithHelp("", "Timer:"), key.WithKeys("")),
				m.keys.toggleTaskTimer,
				m.keys.switchTaskCategory,
				key.NewBinding(key.WithHelp("", ""), key.WithKeys("")),
				key.NewBinding(key.WithHelp("", "Reports:"), key.WithKeys("")),
				// reporting keys
				m.keys.prevReportPage,
				m.keys.nextReportPage,
			},
		}),
	)
}

func (m MyHours) renderInlineHelp() string {
	return m.help.ShortHelpView([]key.Binding{m.keys.openHelp})
}

func (m MyHours) renderNavigation() string {
	cat := m.category(m.settings.DefaultCategoryID)
	var doc strings.Builder
	doc.WriteString(styleNavCap.Render("\uE0BA"))
	doc.WriteString(styleModeIndicator.Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString(styleNavInactive.Render("│"))
	var sections []string
	for i, name := range m.viewNames {
		var style lipgloss.Style
		_, _, isActive := i == 0, i == len(m.viewNames)-1, i == m.state.activeView
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

type viewFunc func(width int, height int) string

func (m MyHours) renderView(view viewFunc) string {
	viewHeight, viewWidth := m.state.viewHeight, m.state.viewWidth
	nav := m.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	viewHeight -= navHeight - 2 // and couple newlines
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, view(viewWidth, viewHeight)))
	doc.WriteString("\n")
	doc.WriteString(lipgloss.Place(viewWidth, navHeight, lipgloss.Center, lipgloss.Center, nav+" "+m.renderInlineHelp()))
	return styleWindow.Render(doc.String())
}

func (m MyHours) renderTimer(width, _ int) string {
	var doc strings.Builder
	cat := m.category(m.state.timerCategoryID)
	w := min(40, width)
	elapsed := m.timer.view()
	started := m.timer.started()
	style := styleTimerContainer.Width(w).BorderForeground(cat.ForegroundColor())
	doc.WriteString(styleTimerLabel.Render("Tracking:"))
	doc.WriteString(lipgloss.NewStyle().Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Elapsed:"))
	doc.WriteString(elapsed)
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Started:"))
	if !started.IsZero() {
		doc.WriteString(started.Format(time.DateTime))
	}
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Task ID:"))
	switch {
	case m.state.activeRecordID > 0:
		doc.WriteString(strconv.FormatInt(m.state.activeRecordID, 10))
	case m.state.previousRecordID > 0:
		doc.WriteString(strconv.FormatInt(m.state.previousRecordID, 10))
	}
	return style.Render(doc.String())
}

func (m MyHours) renderReport(viewWidth, viewHeight int) string {
	if m.state.reportLoading {
		return "Loading ..."
	}
	container := styleReportContainer.Width(viewWidth).Height(viewHeight)
	tableWidth := viewWidth - container.GetHorizontalFrameSize()
	tableHeight := viewHeight - container.GetVerticalFrameSize()
	cat := m.category(m.settings.DefaultCategoryID)
	headers := m.state.reportHeaders
	rows := m.state.reportRows
	styleFunc := m.state.reportStyle
	tbl := table.New().
		Width(tableWidth).
		Height(tableHeight).
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(r, c int) lipgloss.Style {
			if r == -1 {
				return styleFunc(r, c, headers)
			}
			return styleFunc(r, c, rows[r])
		})
	catStyle := lipgloss.NewStyle().Foreground(cat.ForegroundColor())
	var title strings.Builder
	title.WriteString(catStyle.Render(cat.Name))
	title.WriteString(": ")
	title.WriteString(m.state.reportTitle)
	var doc strings.Builder
	doc.WriteString(styleReportTitle.Render(title.String()))
	doc.WriteString("\n")
	doc.WriteString(tbl.Render())
	return container.Render(doc.String())
}
