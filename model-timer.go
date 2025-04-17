package myhours

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newTimerModel(interval time.Duration) timerModel {
	return timerModel{interval: interval}
}

type timerModel struct {
	interval time.Duration
	start    time.Time
	end      time.Time
	tag      int
	running  bool
}

// Init starts the stopwatch.
func (m timerModel) Init() tea.Cmd {
	return nil
}

func (m timerModel) Update(message tea.Msg) (timerModel, tea.Cmd) {
	switch msg := message.(type) {
	case timerStartMsg:
		if m.running {
			return m, nil
		}
		m.start = msg.from
		m.running = true
		return m, nil
	case timerStopMsg:
		// register final tick.
		m.end = msg.end
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

// View of the timer component.
func (m timerModel) View() string {
	switch {
	case m.running:
		return "🕒" + time.Now().Sub(m.start).Truncate(time.Second).String()
	case m.start.IsZero():
		return "😴 Idle..."
	default:
		return "✅" + m.end.Sub(m.start).Truncate(time.Second).String()
	}
}

// StartFrom sets the starting time for the stopwatch to given time.Time.
func (m timerModel) StartFrom(t time.Time) tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return timerStartMsg{from: t}
	}, timerTick(m.tag, m.interval))
}

// Start starts the stopwatch, counting from now.
func (m timerModel) Start() tea.Cmd {
	return tea.Sequence(func() tea.Msg {
		return timerStartMsg{from: time.Now()}
	}, timerTick(m.tag, m.interval))
}

// Stop stops the stopwatch.
func (m timerModel) Stop() tea.Cmd {
	return func() tea.Msg {
		return timerStopMsg{m.start, time.Now()}
	}
}

// Elapsed returns the time elapsed.
func (m timerModel) Elapsed() time.Duration {
	switch {
	case m.running:
		return time.Now().Sub(m.start)
	case m.start.IsZero():
		return time.Duration(0)
	default:
		return m.end.Sub(m.start)
	}
}

// Since returns the starting time of the stopwatch
func (m timerModel) Since() time.Time {
	return m.start
}

func (m timerModel) Running() bool {
	return m.running
}

func timerTick(tag int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return timerTickMsg{tag: tag}
	})
}
