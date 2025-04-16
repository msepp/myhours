package myhours

import (
	"database/sql"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type reHydrateMsg struct {
	recordID int64
	since    time.Time
	category int64
}

type viewRenderer interface {
	Name() string
	Update(app Application, msg tea.Msg) (tea.Model, tea.Cmd)
	View(app Application, viewWidth, viewHeight int) string
	Init(app Application) tea.Cmd
	HelpKeys(keys keymap) []key.Binding
}

// Application is the myhours application handle / model. Implements the application
// logic for time tracking.
type Application struct {
	l               *slog.Logger
	db              *sql.DB
	showHelp        bool
	views           []viewRenderer
	categories      []category
	config          AppConfig
	keymap          keymap
	help            help.Model
	wWidth          int
	wHeight         int
	activeView      int
	defaultCategory int64
	quitting        bool
}

func (app Application) Init() tea.Cmd {
	var commands []tea.Cmd
	// get partial record if one exists, this allows continuing tracking time
	// from where the timer left.
	partial, err := app.partialRecord()
	if err != nil {
		app.l.Warn("failed to reinit partial record", slog.String("error", err.Error()))
	} else if partial != nil {
		commands = append(commands, func() tea.Msg {
			return reHydrateMsg{recordID: partial.ID, since: partial.Start, category: partial.CategoryID}
		})
	}
	// inits from view components
	for _, view := range app.views {
		if cmd := view.Init(app); cmd != nil {
			commands = append(commands, cmd)
		}
	}
	return tea.Batch(commands...)
}

func (app Application) View() string {
	viewWidth := app.wWidth - windowStyle.GetHorizontalFrameSize()
	viewHeight := app.wHeight - windowStyle.GetHorizontalFrameSize()
	if app.showHelp {
		return lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, app.helpView())
	}
	nav := app.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	viewHeight -= navHeight
	viewContent := app.views[app.activeView].View(app, viewWidth, viewHeight)
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight+1, lipgloss.Center, lipgloss.Center, viewContent))
	doc.WriteString("\n")
	doc.WriteString(nav)
	doc.WriteString(" " + app.help.ShortHelpView([]key.Binding{app.keymap.openHelp}))
	return windowStyle.Render(doc.String())
}

var globalHelpKeys = []key.Binding{appKeyMap.switchGlobalCategory, appKeyMap.tabNext, appKeyMap.tabPrev, appKeyMap.quit, appKeyMap.closeHelp}

func (app Application) helpView() string {
	return app.help.FullHelpView([][]key.Binding{
		globalHelpKeys,
		app.views[app.activeView].HelpKeys(app.keymap),
	})
}

func (app Application) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		app.wWidth = msg.Width
		app.wHeight = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.openHelp, app.keymap.closeHelp):
			app.showHelp = !app.showHelp
			app.keymap.closeHelp.SetEnabled(app.showHelp)
			app.keymap.openHelp.SetEnabled(!app.showHelp)
			return app, nil
		case key.Matches(msg, app.keymap.switchGlobalCategory):
			// update the active category in configuration so we can start up
			// with the same category on next load.
			cat := nextCategory(app.categories, app.defaultCategory)
			app.defaultCategory = cat.id
			if err := app.updateConfig("default_category", strconv.FormatInt(app.defaultCategory, 10)); err != nil {
				app.l.Error("failed to update default category", slog.String("error", err.Error()))
			}
			return app.views[app.activeView].Update(app, message)
		case key.Matches(msg, app.keymap.tabNext):
			app.activeView = min(app.activeView+1, len(app.views)-1)
		case key.Matches(msg, app.keymap.tabPrev):
			app.activeView = max(app.activeView-1, 0)
		case key.Matches(msg, app.keymap.quit):
			app.quitting = true
			return app, tea.Quit
		}
		// when help is visible, don't allow keypresses to flow to other views.
		if app.showHelp {
			return app, nil
		}
	}
	return app.views[app.activeView].Update(app, message)
}

func activeCategory(categories []category, activeID int64) category {
	for _, cat := range categories {
		if cat.id == activeID {
			return cat
		}
	}
	return category{id: 0, name: "unknown"}
}

func nextCategory(categories []category, activeID int64) category {
	var idx int
	for idx = 0; idx < len(categories); idx++ {
		if categories[idx].id == activeID {
			break
		}
	}
	if idx >= len(categories)-1 {
		idx = 0
	} else {
		idx++
	}
	return categories[idx]
}
