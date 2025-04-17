package myhours

import (
	"errors"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Record of time spent. A span of time spent on something.
type Record struct {
	// ID of this particular record
	ID int64
	// Start datetime, when recording started.
	Start time.Time
	// End datetime, when recording finished. If zero, record is considered to be
	// active still, but prefer Finished method.
	End time.Time
	// Duration calculated from start & end.
	Duration time.Duration
	// CategoryID defines the category for the recorded time.
	CategoryID int64
	// Notes for this particular record.
	Notes string
}

// Finished returns if the record has been finished
func (r Record) Finished() bool {
	return !r.End.IsZero()
}

// Validate Record for any inconsistencies. Returns error with validation failure
// reason if Record is somehow broken.
func (r Record) Validate() error {
	if r.Start.IsZero() {
		return errors.New("start time must be non-zero")
	}
	if !r.End.IsZero() || r.Duration > 0 {
		if r.End.Sub(r.Start) != r.Duration {
			return errors.New("duration does not match start&end")
		}
	}
	return nil
}

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
