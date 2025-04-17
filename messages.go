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

// reHydrateMsg is sent to select the initial active task
type reHydrateMsg struct {
	recordID int64
	since    time.Time
	category int64
}

// viewAreaSizeMsg reports a change to the view area (usable area for view data)
type viewAreaSizeMsg struct {
	width, height int
}
