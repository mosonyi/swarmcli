package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Update handles all messages for the stacks view.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		l().Infof("[update]: Received Msg with %d entries", len(msg.Stacks))
		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg.Stacks)
		if err != nil {
			l().Errorf("[update] Error computing hash: %v", err)
			return nil
		}
		m.nodeID = msg.NodeID
		m.setStacks(msg.Stacks)
		m.Visible = true
		return tickCmd()

	case TickMsg:
		l().Infof("StacksView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes (this will return either a Msg or the next TickMsg)
		if m.Visible {
			return CheckStacksCmd(m.lastSnapshot, m.nodeID)
		}
		// Continue polling even if not visible
		return tickCmd()

	case RefreshErrorMsg:
		m.Visible = true
		m.List.Viewport.SetContent(fmt.Sprintf("Error refreshing stacks: %v", msg.Err))
		return nil

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		m.List.Viewport.SetContent(m.List.View())
		return nil

	case tea.KeyMsg:
		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		// Enter triggers navigation
		if msg.String() == "i" || msg.String() == "enter" {
			if m.List.Cursor < len(m.List.Filtered) {
				selected := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						ViewName: servicesview.ViewName,
						Payload:  map[string]interface{}{"stackName": selected.Name},
					}
				}
			}
		}
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) setStacks(stacks []docker.StackEntry) {
	l().Infof("StacksView.setStacks: Updating display with %d stacks", len(stacks))

	// Update backing items but do not clobber the current Filtered slice or
	// query/mode. Re-apply the existing filter so the user's current search
	// remains after a refresh.
	m.List.Items = stacks
	m.List.ApplyFilter()

	m.setRenderItem()

	if m.ready {
		// Refresh viewport content so the parent view sees the filtered
		// content immediately.
		m.List.Viewport.SetContent(m.List.View())
		l().Info("StacksView.setStacks: Viewport content updated")
	} else {
		l().Warn("StacksView.setStacks: View not ready yet, skipping viewport update")
	}
}

// After loading stacks, set RenderItem dynamically with correct column width
func (m *Model) setRenderItem() {
	// Compute column width automatically
	m.List.ComputeAndSetColWidth(func(s docker.StackEntry) string {
		return s.Name
	}, 15)

	// Update RenderItem to use computed colWidth
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	m.List.RenderItem = func(s docker.StackEntry, selected bool, colWidth int) string {
		width := m.List.Viewport.Width
		if width <= 0 {
			width = m.width
		}
		if width <= 0 {
			width = 80
		}

		cols := 3
		sepLen := 2
		starts := make([]int, cols)
		for i := 0; i < cols; i++ {
			starts[i] = (i * width) / cols
		}
		colWidths := make([]int, cols)
		for i := 0; i < cols; i++ {
			if i == cols-1 {
				colWidths[i] = width - starts[i]
			} else {
				colWidths[i] = starts[i+1] - starts[i]
			}
			if colWidths[i] < 1 {
				colWidths[i] = 1
			}
		}

		// Prepare texts with truncation where necessary
		// First column reserves two leading spaces for marker
		nameMax := colWidths[0] - 2
		if nameMax < 0 {
			nameMax = 0
		}
		name := s.Name
		if lipgloss.Width(name) > nameMax {
			// Simple rune-aware truncation with ellipsis
			// fallback: use substring based on runes
			runes := []rune(name)
			if len(runes) > nameMax {
				if nameMax > 1 {
					name = string(runes[:nameMax-1]) + "…"
				} else {
					name = string(runes[:nameMax])
				}
			}
		}
		first := fmt.Sprintf("  %s", name)

		svcStr := fmt.Sprintf("%d", s.ServiceCount)
		svcMax := colWidths[1]
		if lipgloss.Width(svcStr) > svcMax {
			svcRunes := []rune(svcStr)
			if len(svcRunes) > svcMax {
				svcStr = string(svcRunes[:svcMax])
			}
		}

		nodeStr := fmt.Sprintf("%d", s.NodeCount)
		nodeMax := colWidths[2]
		if lipgloss.Width(nodeStr) > nodeMax {
			nodeRunes := []rune(nodeStr)
			if len(nodeRunes) > nodeMax {
				nodeStr = string(nodeRunes[:nodeMax])
			}
		}

		sep := strings.Repeat(" ", sepLen)
		col0 := fmt.Sprintf("%-*s", colWidths[0], first)
		col1 := fmt.Sprintf("%-*s", colWidths[1], svcStr)
		col2 := fmt.Sprintf("%-*s", colWidths[2], nodeStr)

		line := col0 + sep + col1 + sep + col2

		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			col0 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[0], first) + sep)
			col1 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[1], svcStr) + sep)
			col2 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[2], nodeStr))
			return col0 + col1 + col2
		}
		return itemStyle.Render(line)
	}
}
