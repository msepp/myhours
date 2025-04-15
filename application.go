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
	"github.com/charmbracelet/lipgloss/table"
	"github.com/msepp/myhours/stopwatch"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

// Application is the myhours application handle / model. Implements the application
// logic for time tracking.
type Application struct {
	wWidth    int
	wHeight   int
	tabs      []string
	activeTab int
	l         *slog.Logger
	db        *sql.DB
	stopwatch stopwatch.Model
	table     *table.Table
	keymap    keymap
	help      help.Model
	quitting  bool
}

type keymap struct {
	nextTab key.Binding
	prevTab key.Binding
	start   key.Binding
	stop    key.Binding
	reset   key.Binding
	quit    key.Binding
}

func (app Application) Init() tea.Cmd {
	app.stopwatch.Init()
	return nil
}

func (app Application) View() string {
	doc := strings.Builder{}
	var renderedTabs []string
	for i, t := range app.tabs {
		var style lipgloss.Style
		_, _, isActive := i == 0, i == len(app.tabs)-1, i == app.activeTab
		if isActive {
			style = activeTabStyle
		} else {
			style = inactiveTabStyle
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}
	row := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n")
	tabContent := ""
	_, th := lipgloss.Size(doc.String())
	switch app.activeTab {
	case 0:
		tabContent = lipgloss.NewStyle().Bold(true).Render(app.stopwatch.View())
	case 1:
		tw := app.wWidth - windowStyle.GetHorizontalFrameSize()
		if tw > 80 {
			tw = 80
		}
		app.table = app.table.Width(tw)
		app.table = app.table.Height(app.wHeight - th - windowStyle.GetHorizontalFrameSize())
		tabContent = baseStyle.Render(app.table.String())
	}
	height := app.wHeight - th - windowStyle.GetVerticalFrameSize()
	width := app.wWidth - windowStyle.GetHorizontalFrameSize()
	doc.WriteString(lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, tabContent))
	doc.WriteString(app.helpView())
	return windowStyle.Render(doc.String())
}

func (app Application) helpView() string {
	return "\n" + app.help.ShortHelpView([]key.Binding{
		app.keymap.start,
		app.keymap.stop,
		app.keymap.reset,
		app.keymap.quit,
	})
}

func (app Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.wWidth = msg.Width
		app.wHeight = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.quit):
			app.quitting = true
			return app, tea.Quit
		case key.Matches(msg, app.keymap.nextTab):
			app.activeTab = min(app.activeTab+1, len(app.tabs)-1)
			if app.activeTab == 1 {
				var rows [][]string
				for _, w := range recordsAsWeeks(app.getRecords()) {
					for _, d := range w.Days {
						notes := "--"
						if len(d.Notes) > 0 {
							notes = "!"
						}
						rows = append(rows, []string{
							d.Date,
							"W" + strconv.Itoa(w.WeekNo),
							d.WeekDay.String()[:3],
							d.Total.Truncate(time.Second).String(),
							notes,
						})
					}
					rows = append(rows, []string{
						strconv.Itoa(w.Year),
						"W" + strconv.Itoa(w.WeekNo),
						"",
						w.Total.Truncate(time.Second).String(),
						"",
					})
				}
				app.table.ClearRows().Rows(rows...)
			}
			return app, nil
		case key.Matches(msg, app.keymap.prevTab):
			app.activeTab = max(app.activeTab-1, 0)
			return app, nil
		case key.Matches(msg, app.keymap.reset):
			return app, app.stopwatch.Reset()
		case key.Matches(msg, app.keymap.start, app.keymap.stop):
			if app.stopwatch.Running() {
				app.stopwatch.Stop()
				t0 := app.stopwatch.Since()
				t1 := t0.Add(app.stopwatch.Elapsed())
				if err := app.insertRecord(t0, t1, 2, "temporary notes"); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
			}
			app.keymap.stop.SetEnabled(!app.stopwatch.Running())
			app.keymap.start.SetEnabled(app.stopwatch.Running())
			return app, app.stopwatch.Toggle()
		}
	}
	var cmd tea.Cmd
	app.stopwatch, cmd = app.stopwatch.Update(msg)
	return app, cmd
}

func tabBorderWithBottom(middle string) lipgloss.Border {
	border := lipgloss.Border{}
	border.Bottom = middle
	return border
}

var (
	inactiveTabBorder = tabBorderWithBottom(" ")
	activeTabBorder   = tabBorderWithBottom("─")
	highlightColor    = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	inactiveTabStyle  = lipgloss.NewStyle().Border(inactiveTabBorder, true).BorderForeground(highlightColor).Padding(0, 0)
	activeTabStyle    = inactiveTabStyle.Border(activeTabBorder, true)
	windowStyle       = lipgloss.NewStyle().BorderForeground(highlightColor).Padding(0, 0).Margin(0, 1).Align(lipgloss.Center, lipgloss.Center).Border(lipgloss.NormalBorder())
)

// Run starts the myhours application using given database and optional options.
func Run(db *sql.DB, options ...Option) error {
	// Setup the application components and key-bindings
	t := table.New().
		Headers("Date", "Week", "Weekday", "Total", "Notes").
		Border(lipgloss.NormalBorder()).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().Padding(0, 1)
			row++
			if row == 8 || (row > 8 && row%8 == 0) {
				s = s.Background(lipgloss.AdaptiveColor{Dark: "#FFF", Light: "#000"}).Foreground(lipgloss.AdaptiveColor{Dark: "#000", Light: "#FFF"})
			}
			if row >= 0 && col == 4 {
				s = s.AlignHorizontal(lipgloss.Center)
			}
			return s
		})

	appModel := Application{
		tabs:      []string{"Active task", "Reporting"},
		db:        db,
		l:         slog.New(slog.DiscardHandler),
		table:     t,
		stopwatch: stopwatch.NewWithInterval(time.Millisecond * 100),
		keymap: keymap{
			nextTab: key.NewBinding(
				key.WithKeys("right", "l", "n", "tab"),
				key.WithHelp("n", "next tab"),
			),
			prevTab: key.NewBinding(
				key.WithKeys("left", "h", "p", "shift+tab"),
				key.WithHelp("n", "previous tab"),
			),
			start: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "start"),
			),
			stop: key.NewBinding(
				key.WithKeys("s"),
				key.WithHelp("s", "stop"),
			),
			reset: key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "reset"),
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
		opt(&appModel)
	}
	// boot-up the bubbletea runtime with our application model.
	if _, err := tea.NewProgram(appModel, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
