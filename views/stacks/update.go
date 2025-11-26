package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages for the stacks view.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	// --- Stacks loaded ---
	case Msg:
		m.setStacks(msg)
		m.Visible = true
		return nil

	// --- Refresh error ---
	case RefreshErrorMsg:
		m.Visible = true
		m.viewport.SetContent(fmt.Sprintf("Error refreshing stacks: %v", msg.Err))
		return nil

	// --- Resize event ---
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.ready = true
		m.viewport.SetContent(m.buildContent())
		return nil

	// --- Keyboard input ---
	case tea.KeyMsg:
		return handleKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

// setStacks updates the stacks and refreshes viewport content.
func (m *Model) setStacks(msg Msg) {
	m.nodeID = msg.NodeID
	m.entries = msg.Stacks

	// Reapply the filter if currently searching, otherwise reset filtered -> entries
	if m.mode == ModeSearching && m.searchQuery != "" {
		m.applyFilter()
	} else {
		m.filtered = m.entries
		m.cursor = 0
	}

	if !m.ready {
		return
	}

	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent())
	m.ensureCursorVisible()
}

func (m *Model) applyFilter() {
	// if no query -> restore full list
	if m.searchQuery == "" {
		m.filtered = m.entries
		// keep cursor valid
		if len(m.filtered) == 0 {
			m.cursor = 0
		} else if m.cursor >= len(m.filtered) {
			m.cursor = 0
		}
		// ensure viewport shows top of filtered list when clearing search
		m.viewport.GotoTop()
		return
	}

	q := strings.ToLower(m.searchQuery)
	var result []docker.StackEntry

	for _, s := range m.entries {
		if strings.Contains(strings.ToLower(s.Name), q) {
			result = append(result, s)
		}
	}

	m.filtered = result

	// clamp cursor safely for the filtered results
	if len(m.filtered) == 0 {
		m.cursor = 0
		// show an empty result quickly
		m.viewport.GotoTop()
		return
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// ensureCursorVisible keeps the cursor in the visible viewport range.
func (m *Model) ensureCursorVisible() {
	h := m.viewport.Height
	if h < 1 {
		h = 1
	}

	if m.cursor < m.viewport.YOffset {
		m.viewport.YOffset = m.cursor
	} else if m.cursor >= m.viewport.YOffset+h {
		m.viewport.YOffset = m.cursor - h + 1
	}
}
