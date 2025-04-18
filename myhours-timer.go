package myhours

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newTimer(interval time.Duration) timer {
	return timer{interval: interval}
}

// timer provides a very simple timer for the application.
type timer struct {
	interval time.Duration
	t0       time.Time
	t1       time.Time
	tag      int
	running  bool
}

// update timer model based on messages.
func (m timer) update(message tea.Msg) (timer, tea.Cmd) {
	switch msg := message.(type) {
	case timerStartMsg:
		if m.running {
			return m, nil
		}
		m.t0 = msg.from
		m.running = true
		return m, nil
	case timerStopMsg:
		// register final tick.
		m.t1 = msg.end
		m.running = false
		return m, nil
	case timerTickMsg:
		if !m.running {
			return m, nil
		}
		// If a tag is set, and it's not the one we expect, reject the message.
		// This prevents the stopwatch from receiving too many messages and
		// thus ticking too fast.
		if msg.tag > 0 && msg.tag != m.tag {
			return m, nil
		}
		m.tag++
		return m, timerTick(m.tag, m.interval)
	}
	return m, nil
}

// view of the timer component.
func (m timer) view() string {
	switch {
	case m.running:
		return "🕒" + time.Now().Sub(m.t0).Truncate(time.Second).String()
	case m.t0.IsZero():
		return "😴 Idle..."
	default:
		return "✅" + m.t1.Sub(m.t0).Truncate(time.Second).String()
	}
}

// startFrom sets the starting time for the stopwatch to given time.Time.
func (m timer) startFrom(t time.Time) tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return timerStartMsg{from: t}
	}, timerTick(m.tag, m.interval))
}

// start starts the stopwatch, counting from now.
func (m timer) start() tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return timerStartMsg{from: time.Now()}
	}, timerTick(m.tag, m.interval))
}

// stop stops the stopwatch.
func (m timer) stop() tea.Cmd {
	return func() tea.Msg {
		return timerStopMsg{m.t0, time.Now()}
	}
}

// started returns the starting time of the stopwatch
func (m timer) started() time.Time {
	return m.t0
}

func timerTick(tag int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return timerTickMsg{tag: tag}
	})
}
