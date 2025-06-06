package myhours

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// renderer is a function signature for a function that renders a contents
// for a view, using the given dimensions as guideline for content sizing.
type renderer func(width int, height int) string

// View renders the current model state.
//
// Provides compatibility with tea.Model.
func (m MyHours) View() string {
	switch {
	case !m.state.ready:
		// If state is not yet ready, just output a loading placeholder.
		// We'll get here again once state is ready.
		return m.renderFullscreen(m.renderLoadingScreen)
	case m.state.showHelp:
		// Help is handled separately, we don't want to have the navigation and
		// all other distractions visible when showing help.
		return m.renderFullscreen(m.renderHelp)
	default:
		// Nothing special going on, select renderer for active view.
		var view renderer
		switch m.state.activeView {
		case 0:
			view = m.renderTimer
		case 1, 2, 3:
			view = m.renderReport
		default:
			view = func(int, int) string { return "you should not get here.." }
		}
		return m.renderWithNavigation(view)
	}
}

// renderLoadingScreen that indicates something is not ready yet, but should be
// soon. Usable as placeholder when waiting for data.
func (m MyHours) renderLoadingScreen(width, height int) string {
	return styleLoader.Height(height).Width(width).Render("Loading ...")
}

// renderHelp for the application. This is the full help for the app, which
// requires a bit more space.
func (m MyHours) renderHelp(width, _ int) string {
	h := m.help
	h.Width = width
	keys := newKeymap()
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
				keys.switchGlobalCategory,
				keys.nextTab,
				keys.prevTab,
				keys.quit,
				keys.fullScreen,
				keys.closeHelp,
			},
			// view specific keys
			{
				// timer view keys
				key.NewBinding(key.WithHelp("", "Timer:"), key.WithKeys("")),
				keys.startRecord,
				keys.newRecord,
				keys.switchTaskCategory,
				key.NewBinding(key.WithHelp("", ""), key.WithKeys("")),
				key.NewBinding(key.WithHelp("", "Reports:"), key.WithKeys("")),
				// reporting keys
				keys.prevReportPage,
				keys.nextReportPage,
			},
		}),
	)
}

// renderHelpHint renders the small help next to navigation. Basically just
// telling what to press to see the actual help.
func (m MyHours) renderHelpHint() string {
	return m.help.ShortHelpView([]key.Binding{m.keys.openHelp})
}

// renderShortHelp renders the short help for a view.
func (m MyHours) renderShortHelp(width int, keys ...key.Binding) string {
	h := help.New()
	h.Styles = styleHelp
	return styleShortHelp.Width(width).Render(h.ShortHelpView(keys))
}

// renderNavigation renders the navigation bar at the bottom of the screen.
func (m MyHours) renderNavigation() string {
	// gather the sections of the navigation based on available view names.
	var sections []string
	for i, name := range m.viewNames {
		var style lipgloss.Style
		// style based on if view is active now or not.
		if i == m.state.activeView {
			name = "\uE617 " + name
			style = styleNavActive
		} else {
			style = styleNavInactive
		}
		sections = append(sections, style.Render(name))
	}
	// current globally selected category shown at the start of the nav to give
	// a clue on what's show in reports.
	cat := findCategory(m.categories, m.settings.DefaultCategoryID)
	// then construct the navigation bar.
	var doc strings.Builder
	doc.WriteString(styleNavCap.Render("\uE0BA"))
	doc.WriteString(styleModeIndicator.Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString(styleNavInactive.Render("│"))
	doc.WriteString(strings.Join(sections, styleNavJoiner.Render("╱")))
	doc.WriteString(styleNavCap.Render("\uE0BC"))
	return doc.String()
}

// renderFullscreen renders the given content full screen, without anything else
// on the screen. Given render function gets the view size adjusted to account
// for any window styling.
func (m MyHours) renderFullscreen(render renderer) string {
	return styleWindow.Render(render(m.state.viewWidth, m.state.viewHeight))
}

// renderWithNavigation is used to render a component view with navigation. Given
// render function gets width/height adjusted to account for the navigation.
func (m MyHours) renderWithNavigation(render renderer) string {
	viewHeight, viewWidth := m.state.viewHeight, m.state.viewWidth
	nav := m.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	contentHeight := viewHeight - navHeight + 1
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, contentHeight, lipgloss.Center, lipgloss.Center, render(viewWidth, viewHeight)))
	doc.WriteString("\n")
	doc.WriteString(lipgloss.Place(viewWidth, navHeight, lipgloss.Center, lipgloss.Bottom, nav+" "+m.renderHelpHint()))
	out := styleWindow.Height(m.state.viewHeight).Width(m.state.viewWidth).Render(doc.String())
	return out
}

func (m MyHours) renderTimer(width, _ int) string {
	// limit the max width to something usable. The view gets really weird after
	// scaling too wide.
	w := min(40, width)
	// get the current timer parameters, we'll render these with labels.
	// elapsed is the time currently spent on the possibly active task. Should
	// show idle or previous task if not running right now.
	elapsed := m.timer.view()
	// started is the time when current/previous task was started.
	started := m.timer.started()
	// we also show the category assigned to the task. If a task ID exists, this
	// still allows swapping the category for it.
	cat := findCategory(m.categories, m.state.activeRecord.CategoryID)
	// now we build the actual view.
	var doc strings.Builder
	doc.WriteString(styleTimerLabel.Render("Tracking:"))
	doc.WriteString(lipgloss.NewStyle().Foreground(cat.ForegroundColor()).Render(cat.Name))
	// display the active/previous ID, depending on which is found. If there's
	// an active record ID, then a record is running right now.
	// If only previous record ID set, then a record was made, but stopped.
	if m.state.activeRecord.ID > 0 {
		doc.WriteString(" (id: ")
		doc.WriteString(strconv.FormatInt(m.state.activeRecord.ID, 10))
		doc.WriteString(")")
	}
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Started:"))
	// if task started is zero, we'll just omit the detail nothing is/has been running.
	if !started.IsZero() {
		doc.WriteString(started.Format(time.DateTime + " -0700"))
	}
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Now:"))
	if !started.IsZero() {
		doc.WriteString(time.Now().Format(time.DateTime + " -0700"))
	}
	doc.WriteString("\n")
	doc.WriteString(styleTimerLabel.Render("Elapsed:"))
	doc.WriteString(elapsed)
	doc.WriteString("\n")
	// Form the container style and render the document into it.
	style := styleTimerContainer.Width(w).BorderForeground(cat.ForegroundColor())
	var box strings.Builder
	box.WriteString(style.Render(doc.String()))
	box.WriteString("\n")
	box.WriteString(m.renderShortHelp(width, m.keys.newRecord, m.keys.startRecord, m.keys.stopRecord))
	return box.String()
}

// renderReport builds a report of how time has been spent for some time window
// and formatting options currently set in application state.
func (m MyHours) renderReport(width, height int) string {
	// while report is loading, just output a loading screen
	if m.state.reportLoading {
		return m.renderLoadingScreen(width, height)
	}
	var (
		// We have to calculate some dimensions for the table to make it fit a bit
		// better and ensure it's getting clipped correctly if needed.
		container   = styleReportContainer.Width(width)
		tableWidth  = width - container.GetHorizontalFrameSize()
		tableHeight = height - container.GetVerticalFrameSize()
		// select currently active category, we'll render it also on top of the table.
		cat      = findCategory(m.categories, m.settings.DefaultCategoryID)
		catStyle = lipgloss.NewStyle().Foreground(cat.ForegroundColor())
	)
	// create the new table.
	tbl := table.New().Width(tableWidth).Height(tableHeight)
	// attach data to it.
	tbl = tbl.Headers(m.state.reportHeaders...).Rows(m.state.reportRows...)
	// add styling instructions. We use are wrapper to have access to the table
	// row data, as we want to style some things based on content.
	tbl = tbl.StyleFunc(tableStyleWrapper(m.state.reportStyle, m.state.reportHeaders, m.state.reportRows))
	// build the table title first.
	var title strings.Builder
	title.WriteString(catStyle.Render(cat.Name))
	title.WriteString(": ")
	title.WriteString(m.state.reportTitle)
	// then build the whole report view content by combining a stylized title
	// and the renderer table.
	var doc strings.Builder
	doc.WriteString(styleReportTitle.Render(title.String()))
	doc.WriteString("\n")
	doc.WriteString(tbl.Render())
	doc.WriteString("\n")
	doc.WriteString(m.renderShortHelp(width, m.keys.prevReportPage, m.keys.nextReportPage))
	return container.Render(doc.String())
}

func tableStyleWrapper(cellStyler func(int, int, []string) lipgloss.Style, headers []string, rows [][]string) func(int, int) lipgloss.Style {
	return func(r, c int) lipgloss.Style {
		if r == -1 {
			return cellStyler(r, c, headers)
		}
		return cellStyler(r, c, rows[r])
	}
}
