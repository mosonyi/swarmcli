package polling

import (
	"swarmcli/core/primitives/hash"
	"time"

	swarmlog "swarmcli/utils/log"

	tea "github.com/charmbracelet/bubbletea"
)

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("polling")
}

// TickMsg represents a polling tick event
type TickMsg time.Time

// PollInterval is the default polling interval (5 seconds)
const PollInterval = 5 * time.Second

// Poller provides generic polling functionality for any view
type Poller[T any] struct {
	lastSnapshot uint64
	interval     time.Duration
	loadFunc     func() ([]T, error)
	msgBuilder   func([]T) tea.Msg
}

// New creates a new Poller for type T
// loadFunc should fetch the latest data
// msgBuilder should create the appropriate message type from the data
func New[T any](loadFunc func() ([]T, error), msgBuilder func([]T) tea.Msg) *Poller[T] {
	return &Poller[T]{
		interval:   PollInterval,
		loadFunc:   loadFunc,
		msgBuilder: msgBuilder,
	}
}

// NewWithInterval creates a new Poller with a custom interval
func NewWithInterval[T any](interval time.Duration, loadFunc func() ([]T, error), msgBuilder func([]T) tea.Msg) *Poller[T] {
	return &Poller[T]{
		interval:   interval,
		loadFunc:   loadFunc,
		msgBuilder: msgBuilder,
	}
}

// TickCmd returns a command that will trigger a tick after the interval
func (p *Poller[T]) TickCmd() tea.Cmd {
	return tea.Tick(p.interval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// CheckCmd checks if data has changed and returns an update message if so
// Otherwise returns a TickMsg to schedule the next poll
func (p *Poller[T]) CheckCmd() tea.Cmd {
	return func() tea.Msg {
		l().Info("Poller: Polling for changes")

		data, err := p.loadFunc()
		if err != nil {
			l().Errorf("Poller: Load failed: %v", err)
			// Schedule next poll even on error
			return p.TickCmd()()
		}

		newHash, err := hash.Compute(data)
		if err != nil {
			l().Errorf("Poller: Load failed: %v", err)
			// Schedule next poll even on error
			return p.TickCmd()()
		}

		if p.lastSnapshot == 0 {
			// First load, no comparison
			l().Infof("Poller: Initial load with %d entries", len(data))
			p.lastSnapshot = newHash
			return p.msgBuilder(data)
		}

		l().Infof("Poller: lastHash=%s, newHash=%s, count=%d",
			hash.Fmt(p.lastSnapshot), hash.Fmt(newHash), len(data))

		// Only return update message if something changed
		if newHash != p.lastSnapshot {
			l().Info("Poller: Change detected! Refreshing data")
			p.lastSnapshot = newHash
			return p.msgBuilder(data)
		}

		l().Info("Poller: No changes detected, scheduling next poll")
		// Schedule next poll
		return p.TickCmd()()
	}
}

// UpdateHash updates the stored hash with new data
// Call this when you receive fresh data from other sources
func (p *Poller[T]) UpdateHash(data []T) {
	p.lastSnapshot, _ = hash.Compute(data)
}

// GetLastHash returns the last computed hash
func (p *Poller[T]) GetLastHash() uint64 {
	return p.lastSnapshot
}
