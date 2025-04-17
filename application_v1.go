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
)

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

func (app ApplicationV1) View() string {
	switch {
	case app.state.showHelp:
		return app.renderHelp()
	default:
		return app.renderView(app.renderTimer)
	}
}

func (app ApplicationV1) renderHelp() string {
	return lipgloss.Place(
		app.state.viewWidth,
		app.state.viewHeight,
		lipgloss.Center,
		lipgloss.Center,
		app.models.help.FullHelpView([][]key.Binding{{
			app.keys.switchGlobalCategory,
			app.keys.tabNext,
			app.keys.tabPrev,
			app.keys.quit,
			app.keys.closeHelp,
		}}),
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

func (app ApplicationV1) renderView(viewRenderer func() string) string {
	viewHeight, viewWidth := app.state.viewHeight, app.state.viewWidth
	nav := app.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	viewHeight -= navHeight - 1 // navigation height + one newline
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, viewRenderer()))
	doc.WriteString("\n")
	doc.WriteString(lipgloss.Place(viewWidth, navHeight, lipgloss.Center, lipgloss.Center, nav+" "+app.renderInlineHelp()))
	return styleWindow.Render(doc.String())
}

func (app ApplicationV1) renderTimer() string {
	var doc strings.Builder
	cat := activeCategory(app.categories, app.state.timerCategoryID)
	w := min(40, app.state.viewWidth)
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

func (app ApplicationV1) activeCategory(id int64) Category {
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
