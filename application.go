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
	wWidth         int
	wHeight        int
	tabs           []string
	activeTab      int
	activeRecordID int64
	activeCategory int64
	offsetWeek     int
	offsetMonth    int
	offsetYear     int
	categories     []category
	config         AppConfig
	l              *slog.Logger
	db             *sql.DB
	stopwatch      stopwatch.Model
	table          *table.Table
	keymap         keymap
	help           help.Model
	quitting       bool
}

type keymap struct {
	category       key.Binding
	tabNext        key.Binding
	tabPrev        key.Binding
	offsetBackward key.Binding
	offsetForward  key.Binding
	start          key.Binding
	stop           key.Binding
	reset          key.Binding
	quit           key.Binding
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
	return catStyle.Render(current.name) + ": " + app.help.ShortHelpView([]key.Binding{
		app.keymap.start,
		app.keymap.stop,
		app.keymap.reset,
		app.keymap.category,
		app.keymap.tabNext,
		app.keymap.tabPrev,
		app.keymap.quit,
	})
}

type ReHydrateMsg struct {
	RecordID int64
	Since    time.Time
	Category int64
}

type SwitchCategoryMsg struct {
	ID int64
}

func (app Application) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case ReHydrateMsg:
		app.activeRecordID = v.RecordID
		app.activeCategory = v.Category
		return app, app.stopwatch.StartFrom(v.Since)
	case tea.WindowSizeMsg:
		app.wWidth = v.Width
		app.wHeight = v.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(v, app.keymap.category):
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
		case key.Matches(v, app.keymap.offsetBackward):
			app.offsetWeek--
			app.offsetMonth--
			app.offsetYear--
			return app.updateTable(), nil
		case key.Matches(v, app.keymap.offsetForward):
			app.offsetWeek = min(app.offsetWeek+1, 0)
			app.offsetMonth = min(app.offsetMonth+1, 0)
			app.offsetYear = min(app.offsetYear+1, 0)
			return app.updateTable(), nil
		case key.Matches(v, app.keymap.tabNext):
			// TODO: pause stopwatch.
			app.activeTab = min(app.activeTab+1, len(app.tabs)-1)
			return app.updateTable(), nil
		case key.Matches(v, app.keymap.tabPrev):
			// TODO: resume stopwatch.
			app.activeTab = max(app.activeTab-1, 0)
			return app.updateTable(), nil
		case key.Matches(v, app.keymap.reset):
			return app, app.stopwatch.Reset(true)
		case key.Matches(v, app.keymap.start, app.keymap.stop):
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
		case key.Matches(v, app.keymap.quit):
			app.quitting = true
			return app, tea.Quit
		}
	}
	var cmd tea.Cmd
	app.stopwatch, cmd = app.stopwatch.Update(msg)
	return app, cmd
}

func (app Application) updateTable() Application {
	var rows [][]string
	switch app.activeTab {
	case 1:
		rows = rowsByWeek(app.getRecords(weekFilter(app.offsetWeek)))
	case 2:
		rows = rowsByMonth(app.getRecords(monthFilter(app.offsetMonth)))
	case 3:
		rows = rowsByYear(app.getRecords(yearFilter(app.offsetYear)))
	}
	app.table = app.table.ClearRows().Rows(rows...)
	return app
}

func rowsByWeek(records []dbRecord) [][]string {
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
			"Total",
			"",
			"",
			w.Total.Truncate(time.Second).String(),
			"",
		})
	}
	return rows
}
func rowsByMonth(records []dbRecord) [][]string {
	var rows [][]string
	for _, m := range recordsAsMonths(records) {
		for _, w := range m.Weeks {
			fd, ld := w.DateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				m.Month.String()[:3],
				"W" + strconv.Itoa(w.WeekNo),
				w.Total.Truncate(time.Second).String(),
				"",
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			"",
			m.Total.Truncate(time.Second).String(),
			"",
		})
	}
	return rows
}

func rowsByYear(records []dbRecord) [][]string {
	var rows [][]string
	for _, y := range recordsAsYears(records) {
		for _, m := range y.Months {
			fd, ld := m.DateRange()
			rows = append(rows, []string{
				fd + " – " + ld,
				m.Month.String(),
				"",
				m.Total.Truncate(time.Second).String(),
				"",
			})
		}
		rows = append(rows, []string{
			"Total",
			"",
			"",
			y.Total.Truncate(time.Second).String(),
			"",
		})
	}
	return rows
}
func tabBorderWithBottom(middle string) lipgloss.Border {
	border := lipgloss.Border{}
	border.Bottom = middle
	return border
}

func weekFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now().AddDate(0, 0, offset*7) // week is always 7 days, so this still works.
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

func monthFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now()
	y, m, _ := now.Date()
	// first day of current month, minus as many months as offset says
	from := time.Date(y, m, 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
	before := time.Date(y, m+1, 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
	return from, before
}

func yearFilter(offset int) (time.Time, time.Time) {
	if offset > 0 {
		offset = 0
	}
	now := time.Now()
	y, _, _ := now.Date()
	// first day of current year, minus as many years as offset says
	from := time.Date(y, 1, 1, 0, 0, 0, 0, time.Local).AddDate(offset, 0, 0)
	before := time.Date(y+1, 1, 1, 0, 0, 0, 0, time.Local).AddDate(0, offset, 0)
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
	app := Application{
		tabs:      []string{"Now", "Weeks", "Months", "Years"},
		db:        db,
		l:         slog.New(slog.DiscardHandler),
		table:     t,
		stopwatch: stopwatch.NewWithInterval(time.Millisecond * 100),
		keymap: keymap{
			category: key.NewBinding(
				key.WithKeys("c"),
				key.WithHelp("c", "category"),
			),
			tabNext: key.NewBinding(
				key.WithKeys("right", "l", "n", "tab"),
				key.WithHelp("n", "next tab"),
			),
			tabPrev: key.NewBinding(
				key.WithKeys("left", "h", "p", "shift+tab"),
				key.WithHelp("n", "previous tab"),
			),
			offsetForward: key.NewBinding(
				key.WithKeys("down", "j"),
				key.WithHelp("j", "next period"),
			),
			offsetBackward: key.NewBinding(
				key.WithKeys("up", "k"),
				key.WithHelp("k", "previous period"),
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
	// get partial record if one exists, this allows continuing tracking time
	// from where the timer left.
	var partial *dbRecord
	if partial, err = app.partialRecord(); err != nil {
		return fmt.Errorf("partialRecord: %w", err)
	}
	// boot-up the bubbletea runtime with our application model.
	prog := tea.NewProgram(app, tea.WithAltScreen())
	if partial != nil {
		go func() {
			prog.Send(ReHydrateMsg{RecordID: partial.ID, Since: partial.Start, Category: partial.CategoryID})
		}()
	}
	if _, err = prog.Run(); err != nil {
		return fmt.Errorf("bubbletea.NewProgram().Run(): %w", err)
	}
	return nil
}
