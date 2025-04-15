package myhours

import (
	"database/sql"
	"fmt"
	"log"
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
	wWidth         int
	wHeight        int
	tabs           []string
	activeTab      int
	activeRecordID int64
	l              *slog.Logger
	db             *sql.DB
	stopwatch      stopwatch.Model
	table          *table.Table
	keymap         keymap
	help           help.Model
	quitting       bool
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
	case 1, 2, 3:
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
		app.keymap.nextTab,
		app.keymap.prevTab,
	})
}

type ReHydrateMsg struct {
	RecordID int64
	Since    time.Time
}

func (app Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ReHydrateMsg:
		app.activeRecordID = msg.RecordID
		return app, app.stopwatch.StartFrom(msg.Since)
	case tea.WindowSizeMsg:
		app.wWidth = msg.Width
		app.wHeight = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, app.keymap.quit):
			app.quitting = true
			return app, tea.Quit
		case key.Matches(msg, app.keymap.nextTab):
			// TODO: pause stopwatch.
			app.activeTab = min(app.activeTab+1, len(app.tabs)-1)
			return app.updateTable(), nil
		case key.Matches(msg, app.keymap.prevTab):
			// TODO: resume stopwatch.
			app.activeTab = max(app.activeTab-1, 0)
			return app.updateTable(), nil
		case key.Matches(msg, app.keymap.reset):
			return app, app.stopwatch.Reset(true)
		case key.Matches(msg, app.keymap.start, app.keymap.stop):
			app.keymap.stop.SetEnabled(!app.stopwatch.Running())
			app.keymap.start.SetEnabled(app.stopwatch.Running())
			switch app.stopwatch.Running() {
			case false:
				start := time.Now()
				var err error
				if app.activeRecordID, err = app.startRecord(start, 2, ""); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				return app, app.stopwatch.StartFrom(start)
			case true:
				now := time.Now()
				if err := app.finishRecord(app.activeRecordID, app.stopwatch.Since(), now, "fake notes"); err != nil {
					app.l.Error("failed to store record", slog.String("error", err.Error()))
				}
				app.activeRecordID = 0
				return app, app.stopwatch.Reset(false)
			}
		}
	}
	var cmd tea.Cmd
	app.stopwatch, cmd = app.stopwatch.Update(msg)
	return app, cmd
}

func (app Application) updateTable() Application {
	var records []dbRecord
	switch app.activeTab {
	case 1:
		records = app.getRecords(currentWeekFilter())
	case 2:
		records = app.getRecords(currentMonthFilter())
	case 3:
		records = app.getRecords(currentYearFilter())
	}
	var rows [][]string
	for _, w := range recordsAsWeeks(records) {
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
	app.table = app.table.ClearRows().Rows(rows...)
	return app
}

func tabBorderWithBottom(middle string) lipgloss.Border {
	border := lipgloss.Border{}
	border.Bottom = middle
	return border
}

func currentWeekFilter() (time.Time, time.Time) {
	now := time.Now()
	wd := int(now.Weekday())
	if wd == 0 {
		wd = 7
	}
	wd--
	y, m, d := now.Date()
	base := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	switch wd {
	case 0:
		return base, base.AddDate(0, 0, 7)
	case 6:
		return base.AddDate(0, 0, -6), base.AddDate(0, 0, 1)
	default:
		return base.AddDate(0, 0, -1*wd), base.AddDate(0, 0, 7-wd)
	}
}

func currentMonthFilter() (time.Time, time.Time) {
	now := time.Now()
	y, m, _ := now.Date()
	from := time.Date(y, m, 1, 0, 0, 0, 0, time.Local)
	before := time.Date(y, m+1, 1, 0, 0, 0, 0, time.Local)
	return from, before
}

func currentYearFilter() (time.Time, time.Time) {
	now := time.Now()
	y, _, _ := now.Date()
	from := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local)
	before := time.Date(y+1, 1, 1, 0, 0, 0, 0, time.Local)
	return from, before
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
		tabs:      []string{"Active task", "Report: week", "Report: month", "Report: year"},
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
	// get partial record if one exists, this allows continuing tracking time
	// from where the timer left.
	partial, err := appModel.partialRecord()
	if err != nil {
		return fmt.Errorf("partialRecord: %w", err)
	}
	// boot-up the bubbletea runtime with our application model.
	prog := tea.NewProgram(appModel, tea.WithAltScreen())
	if partial != nil {
		go func() {
			log.Printf("%v", partial)
			prog.Send(ReHydrateMsg{RecordID: partial.ID, Since: partial.Start})
		}()
	}
	if _, err = prog.Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
