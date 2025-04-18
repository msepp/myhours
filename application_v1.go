package myhours

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

type reportStyleFunc func(row, col int, rowData []string) lipgloss.Style

type appState struct {
	screenWidth      int
	screenHeight     int
	viewWidth        int
	viewHeight       int
	activeView       int
	timerCategoryID  int64
	activeRecordID   int64
	previousRecordID int64
	showHelp         bool
	quitting         bool
	// reporting data fields
	reportLoading bool
	reportPage    []int
	reportTitle   string
	reportHeaders []string
	reportStyle   reportStyleFunc
	reportRows    [][]string
}

type models struct {
	help  help.Model
	timer timerModel
}

type ApplicationV1 struct {
	db         Database
	l          *slog.Logger
	settings   Settings
	categories []Category
	viewNames  []string
	keys       appKeys
	state      appState
	models     models
}

func (app ApplicationV1) Init() tea.Cmd {
	commands := []tea.Cmd{
		func() tea.Msg {
			categories, err := app.db.Categories()
			if err != nil {
				app.l.Error("failed to fetch settings", slog.String("error", err.Error()))
				return tea.Quit()
			}
			return updateCategoriesMsg{categories: categories}
		},
		func() tea.Msg {
			settings, err := app.db.Settings()
			if err != nil {
				app.l.Error("failed to fetch settings", slog.String("error", err.Error()))
				return tea.Quit()
			}
			return updateSettingsMsg{settings: *settings}
		},
		func() tea.Msg {
			record, err := app.db.ActiveRecord()
			if err != nil {
				app.l.Error("failed to fetch settings", slog.String("error", err.Error()))
				return tea.Quit()
			}
			if record == nil {
				app.l.Info("no active record")
				return nil
			}
			return reHydrateMsg{recordID: record.ID, since: record.Start, category: record.CategoryID}
		},
	}
	return tea.Sequence(commands...)
}

func (app ApplicationV1) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := message.(type) {
	case reportDataMsg:
		if app.state.activeView != msg.viewID {
			// view changed already. Not relevant anymore.
			return app, nil
		}
		if app.reportPageNo() != msg.pageNo {
			// page changed already. Not relevant anymore.
			return app, nil
		}
		if app.settings.DefaultCategoryID != msg.categoryID {
			// category changed already. Not relevant anymore
			return app, nil
		}
		app.state.reportRows = msg.rows
		app.state.reportHeaders = msg.headers
		app.state.reportTitle = msg.title
		app.state.reportStyle = msg.style
		app.state.reportLoading = false
	case reHydrateMsg:
		app.state.timerCategoryID = msg.category
		app.state.activeRecordID = msg.recordID
		commands = append(commands, app.models.timer.StartFrom(msg.since))
	case timerCategoryMsg:
		app.state.timerCategoryID = msg.categoryID
	case updateCategoriesMsg:
		app.categories = msg.categories
	case updateSettingsMsg:
		app.settings = msg.settings
		if app.state.timerCategoryID == 0 {
			app.state.timerCategoryID = msg.settings.DefaultCategoryID
		}
	case recordStartMsg:
		app.state.activeRecordID = msg.recordID
		app.state.timerCategoryID = msg.categoryID
	case recordFinishMsg:
		app.state.previousRecordID = msg.recordID
		app.state.activeRecordID = 0
	case timerStartMsg:
		if app.state.activeRecordID == 0 {
			commands = append(commands, app.startNewRecord(msg.from, app.state.timerCategoryID))
		}
	case timerStopMsg:
		if app.state.activeRecordID > 0 {
			commands = append(commands, app.finishActiveRecord(msg.start, msg.end))
		}
	case tea.WindowSizeMsg:
		app.state.screenWidth = msg.Width
		app.state.screenHeight = msg.Height
		app.state.viewWidth = msg.Width - styleWindow.GetHorizontalFrameSize()
		app.state.viewHeight = msg.Height - styleWindow.GetVerticalFrameSize() - 2
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keys.switchTaskCategory):
			commands = append(commands, app.updateTimerCategoryID(app.nextCategoryID(app.state.timerCategoryID)))
		case key.Matches(msg, app.keys.openHelp, app.keys.closeHelp):
			app.state.showHelp = !app.state.showHelp
			app.keys.openHelp.SetEnabled(!app.state.showHelp)
			app.keys.closeHelp.SetEnabled(app.state.showHelp)
		case key.Matches(msg, app.keys.nextTab):
			app.state.activeView++
			if app.state.activeView >= len(app.viewNames) {
				app.state.activeView = 0
			}
			app.state.reportLoading = true
			if cmd := app.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, app.keys.prevTab):
			app.state.activeView--
			if app.state.activeView < 0 {
				app.state.activeView = len(app.viewNames) - 1
			}
			app.state.reportLoading = true
			if cmd := app.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, app.keys.toggleTaskTimer):
			if app.models.timer.Running() {
				commands = append(commands, app.models.timer.Stop())
			} else {
				commands = append(commands, app.models.timer.Start())
			}
		case key.Matches(msg, app.keys.quit):
			app.state.quitting = true
			return app, tea.Quit
		}
	}
	var cmd tea.Cmd
	if app.models.timer, cmd = app.models.timer.Update(message); cmd != nil {
		commands = append(commands, cmd)
	}
	return app, tea.Batch(commands...)
}

func (app ApplicationV1) startNewRecord(start time.Time, categoryID int64) tea.Cmd {
	return func() tea.Msg {
		id, err := app.db.StartRecord(start, categoryID, "")
		if err != nil {
			app.l.Error("failed to store new record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		return recordStartMsg{recordID: id, categoryID: categoryID}
	}
}

func (app ApplicationV1) finishActiveRecord(start, end time.Time) tea.Cmd {
	return func() tea.Msg {
		if err := app.db.FinishRecord(app.state.activeRecordID, start, end, ""); err != nil {
			app.l.Error("failed to update record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		return recordFinishMsg{recordID: app.state.activeRecordID}
	}
}

func (app ApplicationV1) updateTimerCategoryID(id int64) tea.Cmd {
	return func() tea.Msg {
		if app.state.activeRecordID > 0 {
			if err := app.db.UpdateRecordCategory(app.state.activeRecordID, id); err != nil {
				app.l.Error("failed to update active record category", slog.String("error", err.Error()))
			}
		}
		return timerCategoryMsg{categoryID: id}
	}
}

func (app ApplicationV1) updateReportData() tea.Cmd {
	var (
		viewID     = app.state.activeView
		pageNo     = app.reportPageNo()
		categoryID = app.settings.DefaultCategoryID
		mapperFunc func([]Record) [][]string
		dateFunc   func(int) (time.Time, time.Time)
		headerFunc func() []string
		titleFunc  func(int) string
		styleFunc  reportStyleFunc
	)
	switch app.state.activeView {
	case 1: // Weekly
		dateFunc = weeklyFilter
		headerFunc = weeklyReportHeaders
		titleFunc = weeklyReportTitle
		styleFunc = weeklyReportStyle
		mapperFunc = weeklyRows
	case 2: // Monthly
		dateFunc = monthlyFilter
		headerFunc = monthlyReportHeaders
		titleFunc = monthlyReportTitle
		styleFunc = monthlyReportStyle
		mapperFunc = monthlyRows
	case 3: // Yearly
		dateFunc = yearlyFilter
		headerFunc = yearlyReportHeaders
		titleFunc = yearlyReportTitle
		styleFunc = yearlyReportStyle
		mapperFunc = yearlyRows
	default:
		// not a reporting view
		return nil
	}
	return func() tea.Msg {
		from, before := dateFunc(pageNo)
		res, err := app.db.RecordsInCategory(from, before, categoryID)
		if err != nil {
			app.l.Error("failed to fetch records", slog.String("error", err.Error()))
			return tea.Quit()
		}
		rows := mapperFunc(res)
		return reportDataMsg{
			viewID:     viewID,
			pageNo:     pageNo,
			categoryID: categoryID,
			title:      titleFunc(pageNo),
			headers:    headerFunc(),
			rows:       rows,
			style:      styleFunc,
		}
	}
}

func (app ApplicationV1) View() string {
	switch {
	case app.state.showHelp:
		return app.renderHelp()
	default:
		var view viewFunc
		switch app.state.activeView {
		case 0:
			view = app.renderTimer
		case 1, 2, 3:
			view = app.renderReport
		default:
			view = func(int, int) string { return "you should not get here.." }
		}
		return app.renderView(view)
	}
}

func (app ApplicationV1) renderHelp() string {
	h := app.models.help
	h.Width = app.state.viewWidth
	return lipgloss.Place(
		app.state.viewWidth,
		app.state.viewHeight,
		lipgloss.Center,
		lipgloss.Center,
		// The layout here is really hacky with the pseudo keys to create segment
		// titles, but I'm so lazy I can't be bothered to do it right when this
		// works just fine.
		h.FullHelpView([][]key.Binding{
			// global keys
			{
				key.NewBinding(key.WithHelp("", "Global:"), key.WithKeys("")),
				app.keys.switchGlobalCategory,
				app.keys.nextTab,
				app.keys.prevTab,
				app.keys.quit,
				app.keys.closeHelp,
			},
			// view specific keys
			{
				// timer view keys
				key.NewBinding(key.WithHelp("", "Timer:"), key.WithKeys("")),
				app.keys.toggleTaskTimer,
				app.keys.switchTaskCategory,
				key.NewBinding(key.WithHelp("", ""), key.WithKeys("")),
				key.NewBinding(key.WithHelp("", "Reports:"), key.WithKeys("")),
				// reporting keys
				app.keys.prevReportPage,
				app.keys.nextReportPage,
			},
		}),
	)
}

func (app ApplicationV1) renderInlineHelp() string {
	return app.models.help.ShortHelpView([]key.Binding{app.keys.openHelp})
}

func (app ApplicationV1) renderNavigation() string {
	cat := activeCategory(app.categories, app.settings.DefaultCategoryID)
	var doc strings.Builder
	doc.WriteString(styleNavCap.Render("\uE0BA"))
	doc.WriteString(styleModeIndicator.Render("mode:"))
	doc.WriteString(styleModeIndicator.Foreground(cat.ForegroundColor()).Render(cat.Name))
	doc.WriteString(styleNavInactive.Render("│"))
	var sections []string
	for i, name := range app.viewNames {
		var style lipgloss.Style
		_, _, isActive := i == 0, i == len(app.viewNames)-1, i == app.state.activeView
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

func (app ApplicationV1) renderView(view viewFunc) string {
	viewHeight, viewWidth := app.state.viewHeight, app.state.viewWidth
	nav := app.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	viewHeight -= navHeight - 2 // and couple newlines
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, view(viewWidth, viewHeight)))
	doc.WriteString("\n")
	doc.WriteString(lipgloss.Place(viewWidth, navHeight, lipgloss.Center, lipgloss.Center, nav+" "+app.renderInlineHelp()))
	return styleWindow.Render(doc.String())
}

func (app ApplicationV1) renderTimer(width, _ int) string {
	var doc strings.Builder
	cat := activeCategory(app.categories, app.state.timerCategoryID)
	w := min(40, width)
	elapsed := app.models.timer.View()
	started := app.models.timer.Since()
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
	case app.state.activeRecordID > 0:
		doc.WriteString(strconv.FormatInt(app.state.activeRecordID, 10))
	case app.state.previousRecordID > 0:
		doc.WriteString(strconv.FormatInt(app.state.previousRecordID, 10))
	}
	return style.Render(doc.String())
}

func (app ApplicationV1) renderReport(viewWidth, viewHeight int) string {
	if app.state.reportLoading {
		return "... loading ..."
	}
	container := styleReportContainer.Width(viewWidth).Height(viewHeight)
	tableWidth := viewWidth - container.GetHorizontalFrameSize()
	tableHeight := viewHeight - container.GetVerticalFrameSize()
	cat := app.category(app.settings.DefaultCategoryID)
	headers := app.state.reportHeaders
	rows := app.state.reportRows
	styleFunc := app.state.reportStyle
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
	title.WriteString(app.state.reportTitle)
	var doc strings.Builder
	doc.WriteString(styleReportTitle.Render(title.String()))
	doc.WriteString("\n")
	doc.WriteString(tbl.Render())
	return container.Render(doc.String())
}

func (app ApplicationV1) reportPageNo() int {
	if app.state.activeView >= len(app.state.reportPage) {
		return 0
	}
	return app.state.reportPage[app.state.activeView]
}

func (app ApplicationV1) category(id int64) Category {
	for _, cat := range app.categories {
		if cat.ID == id {
			return cat
		}
	}
	return Category{ID: 0, Name: "unknown"}
}

func (app ApplicationV1) nextCategoryID(categoryID int64) int64 {
	var idx int
	for idx = 0; idx < len(app.categories); idx++ {
		if app.categories[idx].ID == categoryID {
			break
		}
	}
	if idx >= len(app.categories)-1 {
		idx = 0
	} else {
		idx++
	}
	return app.categories[idx].ID
}
