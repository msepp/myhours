package myhours

import "github.com/charmbracelet/lipgloss"

// Category of a record. Used to define what the time was spent on.
type Category struct {
	// ID of the category, identifies a single category.
	ID int64
	// Name of the category.
	Name string
	// BackgroundDark is the background color when rendering on a dark terminal.
	BackgroundDark string
	// BackgroundLight is the background color when rendering on a light terminal.
	BackgroundLight string
	// ForegroundDark is the foreground color when rendering on a dark terminal.
	ForegroundDark string
	// ForegroundLight is the foreground color when rendering on a light terminal.
	ForegroundLight string
}

// ForegroundColor returns adaptive color for rendering the category name for example.
func (c Category) ForegroundColor() lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: c.ForegroundLight, Dark: c.ForegroundDark}
}

// BackgroundColor returns adaptive color for rendering the category name for example.
func (c Category) BackgroundColor() lipgloss.AdaptiveColor {
	return lipgloss.AdaptiveColor{Light: c.BackgroundLight, Dark: c.BackgroundDark}
}
