package myhours

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewRenderer interface {
	Init() tea.Cmd
	Update(msg tea.Msg) tea.Cmd
	View() string
	Name() string
	HelpKeys() []key.Binding
}

// Application is the myhours application handle / model. Implements the application
// logic for time tracking.
type Application struct {
	l               *slog.Logger
	db              Database
	showHelp        bool
	views           []viewRenderer
	categories      []Category
	config          Settings
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
	partial, err := app.db.ActiveRecord()
	if err != nil {
		app.l.Warn("failed to reinit partial record", slog.String("error", err.Error()))
	} else if partial != nil {
		commands = append(commands, func() tea.Msg {
			return reHydrateMsg{recordID: partial.ID, since: partial.Start, category: partial.CategoryID}
		})
	}
	// update the initial categories
	commands = append(commands, func() tea.Msg {
		return updateCategoriesMsg{categories: app.categories}
	})
	// update the initial default category
	commands = append(commands, func() tea.Msg {
		return updateDefaultCategoryMsg{categoryID: app.defaultCategory}
	})
	// inits from view components
	for _, view := range app.views {
		if cmd := view.Init(); cmd != nil {
			commands = append(commands, cmd)
		}
	}
	return tea.Batch(commands...)
}

func (app Application) View() string {
	viewWidth := app.wWidth - windowStyle.GetHorizontalFrameSize()
	viewHeight := app.wHeight - windowStyle.GetVerticalFrameSize()
	if app.showHelp {
		return lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, app.helpView())
	}
	nav := app.renderNavigation()
	_, navHeight := lipgloss.Size(nav)
	viewHeight -= navHeight - 1 // navigation height + one newline
	viewContent := app.views[app.activeView].View()
	doc := strings.Builder{}
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, viewContent))
	doc.WriteString("\n")
	doc.WriteString(lipgloss.Place(viewWidth, navHeight, lipgloss.Center, lipgloss.Center, nav+" "+app.help.ShortHelpView([]key.Binding{app.keymap.openHelp})))
	return windowStyle.Render(doc.String())
}

func (app Application) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case tea.WindowSizeMsg:
		app.wWidth = msg.Width
		app.wHeight = msg.Height
		for _, view := range app.views {
			view.Update(viewAreaSizeMsg{height: msg.Height - windowStyle.GetVerticalFrameSize() - 2, width: msg.Width - windowStyle.GetHorizontalFrameSize()})
		}
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
			app.defaultCategory = cat.ID
			if err := app.db.UpdateSetting(SettingDefaultCategory, strconv.FormatInt(app.defaultCategory, 10)); err != nil {
				app.l.Error("failed to update default category", slog.String("error", err.Error()))
			}
			return app, func() tea.Msg { return updateDefaultCategoryMsg{categoryID: app.defaultCategory} }
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
	var cmd tea.Cmd
	cmd = app.views[app.activeView].Update(message)
	return app, cmd
}

var globalHelpKeys = []key.Binding{appKeyMap.switchGlobalCategory, appKeyMap.tabNext, appKeyMap.tabPrev, appKeyMap.quit, appKeyMap.closeHelp}

func (app Application) helpView() string {
	return app.help.FullHelpView([][]key.Binding{
		globalHelpKeys,
		app.views[app.activeView].HelpKeys(),
	})
}

func activeCategory(categories []Category, activeID int64) Category {
	for _, cat := range categories {
		if cat.ID == activeID {
			return cat
		}
	}
	return Category{ID: 0, Name: "unknown"}
}

func nextCategory(categories []Category, activeID int64) Category {
	var idx int
	for idx = 0; idx < len(categories); idx++ {
		if categories[idx].ID == activeID {
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
