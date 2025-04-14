// Package stopwatch implements a simple stopwatch. Adapted from github.com/charmbracelet/bubbles/stopwatch
package stopwatch

import (
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

var lastID int64

func nextID() int {
	return int(atomic.AddInt64(&lastID, 1))
}

// TickMsg is a message that is sent on every timer tick.
type TickMsg struct {
	// ID is the identifier of the stopwatch that sends the message. This makes
	// it possible to determine which stopwatch a tick belongs to when there
	// are multiple stopwatches running.
	//
	// Note, however, that a stopwatch will reject ticks from other
	// stopwatches, so it's safe to flow all TickMsgs through all stopwatches
	// and have them still behave appropriately.
	ID  int
	tag int
}

// StartMsg is sent when the stopwatch should start.
type StartMsg struct {
	ID   int
	from time.Time
}

// StopMsg is sent when the stopwatch should stop.
type StopMsg struct {
	ID int
}

// ResetMsg is sent when the stopwatch should reset.
type ResetMsg struct {
	ID int
}

// Model for the stopwatch component.
type Model struct {
	d       time.Duration
	t0      time.Time
	id      int
	tag     int
	running bool

	// How long to wait before every tick. Defaults to 1 second.
	Interval time.Duration
}

// NewWithInterval creates a new stopwatch with the given timeout and tick
// interval.
func NewWithInterval(interval time.Duration) Model {
	return Model{
		Interval: interval,
		id:       nextID(),
	}
}

// New creates a new stopwatch with 1s interval.
func New() Model {
	return NewWithInterval(time.Second)
}

// ID returns the unique ID of the model.
func (m Model) ID() int {
	return m.id
}

// Init starts the stopwatch.
func (m Model) Init() tea.Cmd {
	return m.Start()
}

// StartFrom sets the starting time for the stopwatch to given time.Time.
func (m Model) StartFrom(t time.Time) tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return StartMsg{ID: m.id, from: t}
	}, tick(m.id, m.tag, m.Interval))
}

// Start starts the stopwatch, counting from now.
func (m Model) Start() tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return StartMsg{ID: m.id, from: time.Now()}
	}, tick(m.id, m.tag, m.Interval))
}

// Stop stops the stopwatch.
func (m Model) Stop() tea.Cmd {
	return func() tea.Msg {
		return StopMsg{ID: m.id}
	}
}

// Toggle stops the stopwatch if it is running and starts it if it is stopped.
func (m Model) Toggle() tea.Cmd {
	if m.Running() {
		return m.Stop()
	}
	return m.Start()
}

// Reset resets the stopwatch to 0.
func (m Model) Reset() tea.Cmd {
	return func() tea.Msg {
		return ResetMsg{ID: m.id}
	}
}

// Running returns true if the stopwatch is running or false if it is stopped.
func (m Model) Running() bool {
	return m.running
}

// Update handles the timer tick.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch val := msg.(type) {
	case StartMsg:
		if val.ID != m.id {
			return m, nil
		}
		m.t0 = val.from
		m.running = true
	case StopMsg:
		if val.ID != m.id {
			return m, nil
		}
		// register final tick.
		m.d = time.Since(m.t0)
		m.running = false
	case ResetMsg:
		if val.ID != m.id {
			return m, nil
		}
		m.t0 = time.Now()
		m.d = 0
		return m, nil
	case TickMsg:
		if !m.running || val.ID != m.id {
			break
		}
		// If a tag is set, and it's not the one we expect, reject the message.
		// This prevents the stopwatch from receiving too many messages and
		// thus ticking too fast.
		if val.tag > 0 && val.tag != m.tag {
			return m, nil
		}
		m.d = time.Since(m.t0)
		m.tag++
		return m, tick(m.id, m.tag, m.Interval)
	}

	return m, nil
}

// Elapsed returns the time elapsed.
func (m Model) Elapsed() time.Duration {
	return m.d
}

// Since returns the starting time of the stopwatch
func (m Model) Since() time.Time {
	return m.t0
}

// View of the timer component.
func (m Model) View() string {
	return m.d.String()
}

func tick(id int, tag int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return TickMsg{ID: id, tag: tag}
	})
}
