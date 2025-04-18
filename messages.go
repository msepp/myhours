package myhours

import "time"

// updateCategoriesMsg updates the set of available categories
type updateCategoriesMsg struct {
	categories []Category
}

// updateDefaultCategoryMsg update the currently selected default category
type updateDefaultCategoryMsg struct {
	categoryID int64
}

// updateSettingsMsg updates current application settings
type updateSettingsMsg struct {
	settings Settings
}

// timerCategoryMsg update the category of timer.
type timerCategoryMsg struct {
	categoryID int64
}

type recordStartMsg struct {
	recordID   int64
	categoryID int64
}

type recordFinishMsg struct {
	recordID int64
}

// reHydrateMsg is sent to select the initial active task
type reHydrateMsg struct {
	recordID int64
	since    time.Time
	category int64
}

// reportDataMessage contains data for reporting table.
type reportDataMsg struct {
	// viewID identifies the target report view. If current state view
	// isn't matching, then someone may have cycled the views very quickly and
	// the data in this message isn't needed anymore.
	viewID     int
	pageNo     int
	categoryID int64
	title      string
	headers    []string
	rows       [][]string
	style      reportStyleFunc
}

// viewAreaSizeMsg reports a change to the view area (usable area for view data)
type viewAreaSizeMsg struct {
	width, height int
}

// timerTickMsg is a message that is sent on every timer timerTick.
type timerTickMsg struct {
	tag int
}

// timerStartMsg is sent when the stopwatch should start.
type timerStartMsg struct {
	from time.Time
}

// timerStopMsg is sent when the stopwatch should stop.
type timerStopMsg struct {
	start time.Time
	end   time.Time
}
