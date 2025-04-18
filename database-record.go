package myhours

import (
	"errors"
	"time"
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
	// CategoryID defines the category for the recorded time.
	CategoryID int64
	// Notes for this particular record.
	Notes string
}

// Finished returns if the record has been finished
func (r Record) Finished() bool {
	return !r.End.IsZero()
}

// Duration of the record. If Start or End is zero, return value is zero.
func (r Record) Duration() time.Duration {
	if r.Start.IsZero() || r.End.IsZero() {
		return 0
	}
	return r.End.Sub(r.Start)
}

// Validate Record for any inconsistencies. Returns error with validation failure
// reason if Record is somehow broken.
func (r Record) Validate() error {
	if r.Start.IsZero() {
		return errors.New("start time must be non-zero")
	}
	return nil
}
