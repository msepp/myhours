package report

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// StyleFunc allow styling individual entries
type StyleFunc func(row, col int, rowData []string) lipgloss.Style

// ReInitMsg swaps the whole report data, including headers and style func.
type ReInitMsg struct {
	headers   []string
	rows      [][]string
	styleFunc StyleFunc
}

// UpdateDataMsg updates the data for the report table
type UpdateDataMsg struct {
	rows [][]string
}

// UpdateSizeMsg sets the report table dimensions
type UpdateSizeMsg struct {
	width  int
	height int
}

// Model of the report.
type Model struct {
	table     *table.Table
	data      [][]string
	headers   []string
	baseStyle lipgloss.Style
}

// New report model.
func New() *Model {
	return &Model{
		table:     table.New(),
		baseStyle: lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240")),
	}
}

// SetSize sets the size of the report table rendition
func (m *Model) SetSize(width, height int) *Model {
	m.table = m.table.Width(width).Height(height)
	return m
}

// SetRows sets the rows of the report
func (m *Model) SetRows(rows [][]string) *Model {
	m.data = rows
	m.table = m.table.ClearRows().Rows(rows...)
	return m
}

// SetHeaders sets the report headers
func (m *Model) SetHeaders(headers []string) *Model {
	m.headers = headers
	m.table = m.table.Headers(headers...)
	return m
}

// SetStyle sets the data base style
func (m *Model) SetStyle(s lipgloss.Style) *Model {
	m.baseStyle = s
	return m
}

// SetTableBorder sets the table border
func (m *Model) SetTableBorder(s lipgloss.Border) *Model {
	m.table = m.table.Border(s)
	return m
}

// SetStyleFunc sets a function that is called per report row/cell. The returned
// style is used as the render style of the cell.
func (m *Model) SetStyleFunc(fn StyleFunc) *Model {
	m.table = m.table.StyleFunc(func(r, c int) lipgloss.Style {
		if r < 0 || len(m.data) == 0 {
			return fn(r, c, m.headers)
		}
		return fn(r, c, m.data[r])
	})
	return m
}

// ReInit the report details
func (m *Model) ReInit(headers []string, rows [][]string, fn StyleFunc) tea.Cmd {
	return func() tea.Msg {
		return ReInitMsg{headers: headers, rows: rows, styleFunc: fn}
	}
}

// UpdateData the report details
func (m *Model) UpdateData(rows [][]string) tea.Cmd {
	return func() tea.Msg {
		return UpdateDataMsg{rows: rows}
	}
}

// UpdateSize the report details
func (m *Model) UpdateSize(width, height int) tea.Cmd {
	return func() tea.Msg {
		return UpdateSizeMsg{width: width, height: height}
	}
}

// Update the model.
func (m *Model) Update(message tea.Msg) (*Model, tea.Cmd) {
	switch msg := message.(type) {
	case UpdateSizeMsg:
		m.table = m.table.Width(msg.width).Height(msg.height)
		return m, nil
	case UpdateDataMsg:
		m.data = msg.rows
		m.table = m.table.ClearRows().Rows(msg.rows...)
		return m, nil
	case ReInitMsg:
		m.headers = msg.headers
		m.data = msg.rows
		m.table = m.table.Headers(msg.headers...).ClearRows().Rows(msg.rows...).StyleFunc(func(r, c int) lipgloss.Style {
			if r < 0 {
				return msg.styleFunc(r, c, m.headers)
			}
			return msg.styleFunc(r, c, m.data[r])
		})
		return m, nil
	}
	return m, nil
}

func (m *Model) Init() tea.Cmd {
	return func() tea.Msg {
		return nil
	}
}

// View returns the view of the model
func (m *Model) View() string {
	return m.baseStyle.Render(m.table.Render())
}
