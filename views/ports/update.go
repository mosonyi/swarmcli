// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"swarmcli/docker"
	"swarmcli/views/view"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.ready = true
		m.updateViewport()
		// Kick off the first probe round immediately.
		return tea.Batch(tickCmd(), m.launchProbeCmd())

	case TickMsg:
		if m.ready {
			if m.deps.Snapshot != nil {
				m.deps.Snapshot.TriggerRefreshIfNeeded()
			}
			// Launch a new probe round if the last one is old enough.
			if time.Since(m.lastProbeAt) >= ProbeInterval && !m.probing {
				m.updateViewport()
				return tea.Batch(tickCmd(), m.launchProbeCmd())
			}
			m.updateViewport()
		}
		return tickCmd()

	case ProbeResultMsg:
		m.setProbeResults(msg.Results)
		if m.ready {
			m.updateViewport()
		}
		return nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			// Manual re-probe on demand.
			if !m.probing {
				return m.launchProbeCmd()
			}
			return nil
		case "esc":
			return func() tea.Msg {
				return view.GoBackMsg{}
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

// launchProbeCmd fans out real TCP/UDP probes to all nodes in a background
// goroutine and delivers a ProbeResultMsg when done. It is safe to call even
// when no snapshot is available yet.
func (m *Model) launchProbeCmd() tea.Cmd {
	m.probeMu.Lock()
	m.probing = true
	m.probeMu.Unlock()

	// Snapshot the node list now (before entering the goroutine) so we don't
	// hold the model lock during network I/O.
	var entries []docker.NodeEntry
	if m.deps.Snapshot != nil {
		snap := m.deps.Snapshot.GetSnapshot()
		if snap != nil {
			entries = snap.ToNodeEntries()
		}
	}

	return func() tea.Msg {
		results := docker.ProbeAllNodes(entries)
		return ProbeResultMsg{Results: results}
	}
}
