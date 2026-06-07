// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"swarmcli/docker"
	"swarmcli/views/helpbar"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	deps     docker.Deps
	viewport viewport.Model
	width    int
	height   int
	ready    bool

	// probe state
	probeMu      sync.RWMutex
	probeResults []docker.NodeProbeResult
	probing      bool // true while a probe round is in-flight
	lastProbeAt  time.Time
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m *Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "r", Desc: "Re-probe"},
		{Key: "esc", Desc: "Back"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

func (m *Model) HasErrors() bool {
	return false
}

// getProbeResults returns a snapshot of the latest probe results, safe to call
// from the render path.
func (m *Model) getProbeResults() []docker.NodeProbeResult {
	m.probeMu.RLock()
	defer m.probeMu.RUnlock()
	out := make([]docker.NodeProbeResult, len(m.probeResults))
	copy(out, m.probeResults)
	return out
}

// setProbeResults stores results delivered by the background probe goroutine.
func (m *Model) setProbeResults(results []docker.NodeProbeResult) {
	m.probeMu.Lock()
	defer m.probeMu.Unlock()
	m.probeResults = results
	m.probing = false
	m.lastProbeAt = time.Now()
}
