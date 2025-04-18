package sqlite

import (
	"fmt"
	"time"

	"github.com/msepp/myhours"
)

// val returns the value of any pointer, or the zero value of the type if pointer
// is nil.
func val[T any](p *T) T {
	if p == nil {
		return *new(T)
	}
	return *p
}

// ptrNonZero returns a pointer a copy of given value or nil if value is the
// zero value of the type.
func ptrNonZero[T comparable](p T) *T {
	if *new(T) == p {
		return nil
	}
	return &p
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func scanRecord(row scanner) (*myhours.Record, error) {
	var (
		id                   int64
		start                string
		end, duration, notes *string
		categoryID           int64
	)
	if err := row.Scan(&id, &start, &end, &duration, &categoryID, &notes); err != nil {
		return nil, fmt.Errorf("row.Scan: %w", err)
	}
	record := myhours.Record{ID: id, CategoryID: categoryID, Notes: val(notes)}
	var err error
	if record.Start, err = time.Parse(time.RFC3339Nano, start); err != nil {
		return nil, fmt.Errorf("time.Parse(start): %w", err)
	}
	if end != nil {
		if record.End, err = time.Parse(time.RFC3339Nano, *end); err != nil {
			return nil, fmt.Errorf("time.Parse(end): %w", err)
		}
	}
	if duration != nil {
		if record.Duration, err = time.ParseDuration(*duration); err != nil {
			return nil, fmt.Errorf("time.ParseDuration(duration): %w", err)
		}
	}
	return &record, nil
}
