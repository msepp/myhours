package myhours

import (
	"database/sql"
	"fmt"
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
	ShortHelpKeys(keys keymap) []key.Binding
}

type keymap struct {
	category     key.Binding
	tabNext      key.Binding
	tabPrev      key.Binding
	previousPage key.Binding
	nextPage     key.Binding
	start        key.Binding
	stop         key.Binding
	quit         key.Binding
}

// Application is the myhours application handle / model. Implements the application
// logic for time tracking.
type Application struct {
	l              *slog.Logger
	db             *sql.DB
	views          []viewRenderer
	categories     []category
	config         AppConfig
	keymap         keymap
	help           help.Model
	wWidth         int
	wHeight        int
	activeView     int
	activeRecordID int64
	activeCategory int64
	quitting       bool
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
	doc := strings.Builder{}
	doc.WriteString(app.renderNavigation())
	doc.WriteString("\n")
	_, headerHeight := lipgloss.Size(doc.String())
	viewWidth := app.wWidth - windowStyle.GetHorizontalFrameSize()
	viewHeight := app.wHeight - headerHeight - windowStyle.GetHorizontalFrameSize()
	viewContent := app.views[app.activeView].View(app, viewWidth, viewHeight)
	doc.WriteString(lipgloss.Place(viewWidth, viewHeight, lipgloss.Center, lipgloss.Center, viewContent))
	doc.WriteString("\n" + app.helpView())
	return windowStyle.Render(doc.String())
}

func (app Application) helpView() string {
	current := category{id: app.activeCategory, name: "unknown"}
	for _, cat := range app.categories {
		if cat.id == app.activeCategory {
			current = cat
			break
		}
	}
	catStyle := lipgloss.NewStyle().
		Background(lipgloss.AdaptiveColor{Dark: current.bgColorDark, Light: current.bgColorLight}).
		Foreground(lipgloss.AdaptiveColor{Dark: current.fgColorDark, Light: current.fgColorLight}).Padding(0, 1)
	helpKeys := append(app.views[app.activeView].ShortHelpKeys(app.keymap),
		app.keymap.category,
		app.keymap.tabNext,
		app.keymap.tabPrev,
		app.keymap.quit,
	)
	return catStyle.Render(current.name) + ": " + app.help.ShortHelpView(helpKeys)
}

func (app Application) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := message.(type) {
	case reHydrateMsg:
		app.activeRecordID = msg.recordID
		app.activeCategory = msg.category
	case tea.WindowSizeMsg:
		app.wWidth = msg.Width
		app.wHeight = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.category):
			if app.activeCategory == 3 {
				app.activeCategory = 1
			} else {
				app.activeCategory++
			}
			if app.activeRecordID > 0 {
				if err := app.setRecordCategory(app.activeRecordID, app.activeCategory); err != nil {
					app.l.Error("failed to update active record category", slog.String("error", err.Error()))
				}
			}
			if err := app.updateConfig("default_category", strconv.FormatInt(app.activeCategory, 10)); err != nil {
				app.l.Error("failed to update default category", slog.String("error", err.Error()))
			}
			return app, nil
		case key.Matches(msg, app.keymap.tabNext):
			app.activeView = min(app.activeView+1, len(app.views)-1)
		case key.Matches(msg, app.keymap.tabPrev):
			app.activeView = max(app.activeView-1, 0)
		case key.Matches(msg, app.keymap.quit):
			app.quitting = true
			return app, tea.Quit
		}
	}
	return app.views[app.activeView].Update(app, message)
}

// Run starts the myhours application using given database and optional options.
func Run(db *sql.DB, options ...Option) error {
	app := Application{
		db: db,
		l:  slog.New(slog.DiscardHandler),
		views: []viewRenderer{
			newTimerView(time.Millisecond * 100),
			newWeeklyReportView(),
			newMonthlyReportView(),
			newYearlyReportView(),
		},
		keymap: keymap{
			category: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "category"),
			),
			tabNext: key.NewBinding(
				key.WithKeys("right", "l", "n"),
				key.WithHelp("n", "next view"),
			),
			tabPrev: key.NewBinding(
				key.WithKeys("left", "h", "p"),
				key.WithHelp("h", "prev view"),
			),
			nextPage: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("j", "page down"),
			),
			previousPage: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("k", "page up"),
			),
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
			),
			quit: key.NewBinding(
				key.WithKeys("ctrl+c", "q"),
				key.WithHelp("q", "quit"),
			),
		},
		help: help.New(),
	}
	// apply options to customize the application.
	for _, opt := range options {
		opt(&app)
	}
	// fetch category options
	var err error
	if app.categories, err = app.getCategories(); err != nil {
		return fmt.Errorf("load categories: %w", err)
	}
	// load configuration
	if app.config, err = app.loadConfig(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	app.activeCategory = app.config.DefaultCategory
	// boot-up the bubbletea runtime with our application model.
	prog := tea.NewProgram(app, tea.WithAltScreen())
	if _, err = prog.Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
