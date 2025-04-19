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
	// process all messages in a big switch: each message should need only one
	// type of processing, they don't have multiple meanings.
	// We called all side effects into a slice of new commands, that are returned
	// to the bubbletea runtime, to be scheduled and sent back to this Update
	// function at some point.
	//
	// It's maybe worth noting as well, the most of the time we'll let the messages
	// pass down from this switch into the submodels as well (namely the timer),
	// but there are instances where there is no point in processing the message
	// further and an immediate returns is done instead.
	var commands []tea.Cmd
	switch msg := message.(type) {
	case reportDataMsg:
		// new report data is ready. Let's see if we still needed, and if we do,
		// store it to state.
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
		// timer has been initialized. If init contains details for a record, set
		// that record as the active one and start the timer running from record
		// starting timestamp.
		m.state.activeRecord = msg.record
		// if the record has no valid category, set to what ever is the default
		if m.state.activeRecord.CategoryID == 0 {
			m.state.activeRecord.CategoryID = m.settings.DefaultCategoryID
		}
		// if the record is supposed to be active, we must start timer.
		if m.state.activeRecord.Active() {
			commands = append(commands, m.timer.startFrom(m.state.activeRecord.Start))
		}
		// enable keys for default view now that everything should be ready.
		m.keys.openHelp.SetEnabled(true)
		m.keys.nextTab.SetEnabled(true)
		m.keys.prevTab.SetEnabled(true)
		m.keys.switchGlobalCategory.SetEnabled(true)
		m.keys.switchTaskCategory.SetEnabled(true)
		m.keys.stopRecord.SetEnabled(m.state.activeRecord.Active())
		m.keys.startRecord.SetEnabled(!m.state.activeRecord.Active())
		m.keys.newRecord.SetEnabled(!m.state.activeRecord.Active())
		m.state.ready = true
	case updateCategoriesMsg:
		// details for available categories has changed. This pretty much happens
		// only at app init (for now)
		m.categories = msg.categories
	case updateSettingsMsg:
		// settings have been updated. There's two ways this happens:
		// - at application init
		// - when global category setting was changed. It triggers a settings
		//   update.
		m.settings = msg.settings
		// If there's no details for active record, let's swap the task category
		// as well as convenience.
		if m.state.activeRecord.ID == 0 {
			m.state.activeRecord.CategoryID = msg.settings.DefaultCategoryID
		}
		// we must request update of report data, since category affects what is
		// show in the tables. If there's a need to load new stuff, cmd is non-nil.
		if cmd := m.updateReportData(); cmd != nil {
			commands = append(commands, cmd)
		}
	case updateRecordMsg:
		// Record status had been updated.
		m.state.activeRecord = msg.record
		// while record is active, can't start new one.
		m.keys.newRecord.SetEnabled(!m.state.activeRecord.Active())
		m.keys.startRecord.SetEnabled(!m.state.activeRecord.Active())
		m.keys.stopRecord.SetEnabled(m.state.activeRecord.Active())
	case timerStartMsg:
		// timer has started. start a new record in database with the starting
		// timestamp of the timer. But only allow it when the task has no ID yet.
		if m.state.activeRecord.ID == 0 {
			commands = append(commands, m.startNewRecord(msg.from, m.state.activeRecord.CategoryID))
		} else {
			record := m.state.activeRecord
			record.End = time.Time{}
			commands = append(commands, m.updateRecord(record))
		}
	case timerStopMsg:
		// timer has been stopped, we need to finish the currently active record
		// in database with the start/end times recorded by the timer.
		if !m.state.activeRecord.Finished() {
			record := m.state.activeRecord
			record.Start = msg.start
			record.End = msg.end
			commands = append(commands, m.updateRecord(record))
		}
	case timerResetMsg:
		// on timer reset, we reset the record as well.
		record := Record{CategoryID: m.state.activeRecord.CategoryID}
		commands = append(commands, m.updateRecord(record))
	case tea.WindowSizeMsg:
		// window size has changed. Calculate the dimensions of the view usable
		// area after all window dressing. This is used to contain the contents
		// when rending views like reports.
		m.state.screenWidth = msg.Width
		m.state.screenHeight = msg.Height
		m.state.viewWidth = msg.Width - styleWindow.GetHorizontalFrameSize()
		m.state.viewHeight = msg.Height - styleWindow.GetVerticalFrameSize() - 2
	case tea.KeyMsg:
		// some keypress event has happened. We try to avoid doing direct state
		// manipulation here and instead just trigger the side effects that we
		// want. This should keep the update method faster, offloading things like
		// database operations into asynchronous functions.
		switch {
		case key.Matches(msg, m.keys.switchTaskCategory):
			record := m.state.activeRecord
			record.CategoryID = nextCategoryID(m.categories, record.CategoryID)
			commands = append(commands, m.updateRecord(record))
		case key.Matches(msg, m.keys.switchGlobalCategory):
			// switching the global category is based on stored default category
			// setting.
			next := nextCategoryID(m.categories, m.settings.DefaultCategoryID)
			commands = append(commands, m.updateGlobalCategoryID(next))
		case key.Matches(msg, m.keys.openHelp, m.keys.closeHelp):
			m.state.showHelp = !m.state.showHelp
			// when help changes state, we disable/enable the show/close help
			// keys as inverse of one another. This is because we re-use the same
			// key for both operations.
			m.keys.openHelp.SetEnabled(!m.state.showHelp)
			m.keys.closeHelp.SetEnabled(m.state.showHelp)
		case key.Matches(msg, m.keys.nextReportPage):
			// report page change requested. This should trigger re-fetching of
			// data if page actually changed. Max page number is zero (latest).
			var pageNo int
			if pageNo = m.reportPageNo(); pageNo == 0 {
				return m, nil
			}
			// increment page number, up to maximum. This should take care of
			// seemingly impossible situation where pageNo would be positive non-zero.
			m.state.reportPage[m.state.activeView] = incMax(pageNo, 0)
			// request the update of report data.
			if cmd := m.updateReportData(); cmd != nil {
				m.state.reportLoading = true
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.prevReportPage):
			// report page change requested. This should trigger re-fetching of
			// data if page actually changed.
			// decrement page number by one for the previous page, or use max if
			// value is somehow positive non-zero (which it should never be)
			m.state.reportPage[m.state.activeView] = decMax(m.reportPageNo(), 0)
			// and re-request report data update.
			if cmd := m.updateReportData(); cmd != nil {
				m.state.reportLoading = true
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.nextTab):
			// select next active tab. We allow wrapping back to start.
			m.state.activeView = incWrap(m.state.activeView, 0, len(m.viewNames)-1)
			// enable/disable keys for report activities based on if view is
			// currently a reporting view or not.
			m.keys.nextReportPage.SetEnabled(m.state.activeView > 0)
			m.keys.prevReportPage.SetEnabled(m.state.activeView > 0)
			// update report data if reporting view changed / came into view.
			if cmd := m.updateReportData(); cmd != nil {
				m.state.reportLoading = true
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.prevTab):
			// select previous tab. Allow wrapping straight to last.
			m.state.activeView = decWrap(m.state.activeView, 0, len(m.viewNames)-1)
			// enable/disable keys for report activities based on if view is
			// currently a reporting view or not.
			m.keys.nextReportPage.SetEnabled(m.state.activeView > 0)
			m.keys.prevReportPage.SetEnabled(m.state.activeView > 0)
			// update report data if reporting view changed / came into view.
			if cmd := m.updateReportData(); cmd != nil {
				m.state.reportLoading = true
				commands = append(commands, cmd)
			}
		case key.Matches(msg, m.keys.startRecord, m.keys.stopRecord):
			// timer start/stop requested. start or stop based on the current
			// record status
			switch {
			case m.state.activeRecord.Active():
				commands = append(commands, m.timer.stop())
			case m.state.activeRecord.Finished():
				commands = append(commands, m.timer.startFrom(m.state.activeRecord.Start))
			default:
				commands = append(commands, m.timer.start())
			}
		case key.Matches(msg, m.keys.newRecord):
			if !m.state.activeRecord.Active() {
				commands = append(commands, m.timer.reset())
			}
		case key.Matches(msg, m.keys.quit):
			m.state.quitting = true
			return m, tea.Quit
		}
	}
	// If we got this far, we can pass the message also the submodels for triggering
	// what ever changes are needed.
	var cmd tea.Cmd
	if m.timer, cmd = m.timer.update(message); cmd != nil {
		commands = append(commands, cmd)
	}
	// return a batch of changes. This will result in the commands being handled
	// in asynchronous, non-deterministic order. But that should not be an issue
	// for us.
	return m, tea.Batch(commands...)
}

func (m MyHours) startNewRecord(start time.Time, categoryID int64) tea.Cmd {
	return func() tea.Msg {
		id, err := m.db.StartRecord(start, categoryID, "")
		if err != nil {
			m.l.Error("failed to store new record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		var record *Record
		if record, err = m.db.Record(id); err != nil {
			m.l.Error("failed to load record", slog.String("error", err.Error()))
			return tea.Quit()
		}
		return updateRecordMsg{record: *record}
	}
}

func (m MyHours) updateRecord(record Record) tea.Cmd {
	return func() tea.Msg {
		if record.ID > 0 {
			if err := m.db.UpdateRecord(record.ID, record.CategoryID, record.Start, record.End, record.Notes); err != nil {
				m.l.Error("failed to update active record category", slog.String("error", err.Error()))
				return tea.Quit()
			}
		}
		return updateRecordMsg{record: record}
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

// reportPageNo returns the active page number for a report view.
func (m MyHours) reportPageNo() int {
	return indexOrZero(m.state.reportPage, m.state.activeView)
}
