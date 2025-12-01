package polling

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	lastSnapshot string
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

		newHash := computeHash(data)

		if p.lastSnapshot == "" {
			// First load, no comparison
			l().Infof("Poller: Initial load with %d entries", len(data))
			p.lastSnapshot = newHash
			return p.msgBuilder(data)
		}

		l().Infof("Poller: lastHash=%s, newHash=%s, count=%d",
			p.lastSnapshot[:8], newHash[:8], len(data))

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
	p.lastSnapshot = computeHash(data)
}

// GetLastHash returns the last computed hash
func (p *Poller[T]) GetLastHash() string {
	return p.lastSnapshot
}

// computeHash creates a SHA256 hash of the data for change detection
func computeHash[T any](data []T) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		// Fallback to simple string conversion
		return fmt.Sprintf("%v", data)
	}
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash)
}
