package myhours

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Update model state based on the incoming message.
//
// Returns the updated model (MyHours) and command that needs to be executed
// next. Note that the returned command is always the result of tea.Batch, meaning
// multiple commands may be executed as result.
//
// Provides compatibility with tea.Model.
func (m MyHours) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var commands []tea.Cmd
	switch msg := message.(type) {
	case reportDataMsg:
		if m.state.activeView != msg.viewID {
			// view changed already. Not relevant anymore.
			return m, nil
		}
		if m.reportPageNo() != msg.pageNo {
			// page changed already. Not relevant anymore.
			return m, nil
		}
		if m.settings.DefaultCategoryID != msg.categoryID {
			// category changed already. Not relevant anymore
			return m, nil
		}
		m.state.reportRows = msg.rows
		m.state.reportHeaders = msg.headers
		m.state.reportTitle = msg.title
		m.state.reportStyle = msg.style
		m.state.reportLoading = false
	case initTimerMsg:
		if msg.recordID > 0 {
			m.state.timerCategoryID = msg.category
			m.state.activeRecordID = msg.recordID
			commands = append(commands, m.timer.startFrom(msg.since))
		}
		// enable keys for default view now that everything should be ready.
		m.keys.openHelp.SetEnabled(true)
		m.keys.nextTab.SetEnabled(true)
		m.keys.prevTab.SetEnabled(true)
		m.keys.switchGlobalCategory.SetEnabled(true)
		m.keys.switchTaskCategory.SetEnabled(true)
		m.keys.toggleTaskTimer.SetEnabled(true)
		m.state.ready = true
	case timerCategoryMsg:
		m.state.timerCategoryID = msg.categoryID
	case updateCategoriesMsg:
		m.categories = msg.categories
	case updateSettingsMsg:
		m.settings = msg.settings
		// If there's no details of a record, let's swap the task category as well
		// as convenience.
		if m.state.activeRecordID == 0 || m.state.previousRecordID == 0 {
			m.state.timerCategoryID = msg.settings.DefaultCategoryID
		}
		if cmd := m.updateReportData(); cmd != nil {
			commands = append(commands, cmd)
		}
	case recordStartMsg:
		m.state.activeRecordID = msg.recordID
		m.state.timerCategoryID = msg.categoryID
	case recordFinishMsg:
		m.state.previousRecordID = msg.recordID
		m.state.activeRecordID = 0
	case timerStartMsg:
		if m.state.activeRecordID == 0 {
			commands = append(commands, m.startNewRecord(msg.from, m.state.timerCategoryID))
		}
	case timerStopMsg:
		if m.state.activeRecordID > 0 {
			commands = append(commands, m.finishActiveRecord(msg.start, msg.end))
		}
	case tea.WindowSizeMsg:
		m.state.screenWidth = msg.Width
		m.state.screenHeight = msg.Height
		m.state.viewWidth = msg.Width - styleWindow.GetHorizontalFrameSize()
		m.state.viewHeight = msg.Height - styleWindow.GetVerticalFrameSize() - 2
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.switchTaskCategory):
			commands = append(commands, m.updateTimerCategoryID(m.nextCategoryID(m.state.timerCategoryID)))
		case key.Matches(msg, m.keys.switchGlobalCategory):
			commands = append(commands, m.updateGlobalCategoryID(m.nextCategoryID(m.settings.DefaultCategoryID)))
		case key.Matches(msg, m.keys.openHelp, m.keys.closeHelp):
			m.state.showHelp = !m.state.showHelp
			m.keys.openHelp.SetEnabled(!m.state.showHelp)
			m.keys.closeHelp.SetEnabled(m.state.showHelp)
		case key.Matches(msg, m.keys.nextReportPage):
			pageNo := m.reportPageNo() + 1
			if pageNo > 0 {
				pageNo = 0
			}
			m.state.reportLoading = true
			m.state.reportPage[m.state.activeView] = pageNo
			if cmd := m.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.prevReportPage):
			pageNo := m.reportPageNo() - 1
			m.state.reportLoading = true
			m.state.reportPage[m.state.activeView] = pageNo
			if cmd := m.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.nextTab):
			m.state.activeView++
			if m.state.activeView >= len(m.viewNames) {
				m.state.activeView = 0
			}
			m.state.reportLoading = true
			m.keys.nextReportPage.SetEnabled(m.state.activeView > 0)
			m.keys.prevReportPage.SetEnabled(m.state.activeView > 0)
			if cmd := m.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.prevTab):
			m.state.activeView--
			if m.state.activeView < 0 {
				m.state.activeView = len(m.viewNames) - 1
			}
			m.state.reportLoading = true
			m.keys.nextReportPage.SetEnabled(m.state.activeView > 0)
			m.keys.prevReportPage.SetEnabled(m.state.activeView > 0)
			if cmd := m.updateReportData(); cmd != nil {
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.toggleTaskTimer):
			if m.timer.running {
				commands = append(commands, m.timer.stop())
			} else {
				commands = append(commands, m.timer.start())
			}
		case key.Matches(msg, m.keys.quit):
			m.state.quitting = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	if m.timer, cmd = m.timer.update(message); cmd != nil {
		commands = append(commands, cmd)
	}
	return m, tea.Batch(commands...)
}

func (m MyHours) startNewRecord(start time.Time, categoryID int64) tea.Cmd {
	return func() tea.Msg {
		id, err := m.db.StartRecord(start, categoryID, "")
		if err != nil {
			m.l.Error("failed to store new record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		return recordStartMsg{recordID: id, categoryID: categoryID}
	}
}

func (m MyHours) finishActiveRecord(start, end time.Time) tea.Cmd {
	return func() tea.Msg {
		if err := m.db.FinishRecord(m.state.activeRecordID, start, end, ""); err != nil {
			m.l.Error("failed to update record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		return recordFinishMsg{recordID: m.state.activeRecordID}
	}
}

func (m MyHours) updateTimerCategoryID(id int64) tea.Cmd {
	return func() tea.Msg {
		if m.state.activeRecordID > 0 {
			if err := m.db.UpdateRecordCategory(m.state.activeRecordID, id); err != nil {
				m.l.Error("failed to update active record category", slog.String("error", err.Error()))
				return tea.Quit()
			}
		}
		return timerCategoryMsg{categoryID: id}
	}
}

func (m MyHours) updateGlobalCategoryID(id int64) tea.Cmd {
	settings := m.settings
	return func() tea.Msg {
		if err := m.db.UpdateSetting(SettingDefaultCategory, strconv.FormatInt(id, 10)); err != nil {
			m.l.Error("failed to update global category setting", slog.String("error", err.Error()))
			return tea.Quit()
		}
		settings.DefaultCategoryID = id
		return updateSettingsMsg{settings: settings}
	}
}

func (m MyHours) updateReportData() tea.Cmd {
	var (
		r          report
		viewID     = m.state.activeView
		pageNo     = m.reportPageNo()
		categoryID = m.settings.DefaultCategoryID
	)
	switch m.state.activeView {
	case 1: // Weekly
		r = reportWeekly
	case 2: // Monthly
		r = reportMonthly
	case 3: // Yearly
		r = reportYearly
	default:
		// not a reporting view
		return nil
	}
	return func() tea.Msg {
		from, before := r.dates(pageNo)
		res, err := m.db.RecordsInCategory(from, before, categoryID)
		if err != nil {
			m.l.Error("failed to fetch records", slog.String("error", err.Error()))
			return tea.Quit()
		}
		rows := r.mapper(res)
		return reportDataMsg{
			viewID:     viewID,
			pageNo:     pageNo,
			categoryID: categoryID,
			title:      r.title(pageNo),
			headers:    r.headers(),
			rows:       rows,
			style:      r.styles,
		}
	}
}
