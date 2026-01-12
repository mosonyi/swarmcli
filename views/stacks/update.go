package stacksview

import (
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	helpview "swarmcli/views/help"
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
		m.width = msg.Width
		m.height = msg.Height
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true

		// On first resize (initialization), always reset YOffset to 0
		// This fixes the issue where the view is created with small dimensions,
		// then resized, causing YOffset to be incorrectly set
		if m.firstResize {
			m.List.Viewport.YOffset = 0
			m.firstResize = false
			l().Info("First WindowSizeMsg: resetting YOffset to 0")
		} else if m.List.Cursor == 0 {
			// On subsequent resizes, only reset YOffset if cursor is at top
			m.List.Viewport.YOffset = 0
		}
		return nil

	case tea.KeyMsg:
		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		// If ESC is pressed and there's an active filter, clear it instead of quitting
		if msg.Type == tea.KeyEsc && m.List.Query != "" {
			m.List.Query = ""
			m.List.Mode = filterlist.ModeNormal
			m.List.ApplyFilter()
			m.List.Cursor = 0
			m.List.Viewport.GotoTop()
			return nil
		}

		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		// Show help screen
		if msg.String() == "?" {
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: "help",
					Payload:  GetStacksHelpContent(),
				}
			}
		}

		// Enter triggers navigation to services
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

		// 'p' shows tasks for selected stack
		if msg.String() == "p" {
			if m.List.Cursor < len(m.List.Filtered) {
				selected := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						ViewName: "tasks",
						Payload:  selected.Name,
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

	// Preserve filter query and cursor position
	oldQuery := m.List.Query
	oldMode := m.List.Mode
	oldCursor := m.List.Cursor

	m.List.Items = stacks

	// Restore filter query and mode
	m.List.Query = oldQuery
	m.List.Mode = oldMode

	// Re-apply filter to update filtered list
	if oldQuery != "" {
		m.List.ApplyFilter()
	} else {
		m.List.Filtered = stacks
	}

	// Restore cursor position if still valid
	if oldCursor < len(m.List.Filtered) {
		m.List.Cursor = oldCursor
	} else if len(m.List.Filtered) > 0 {
		m.List.Cursor = len(m.List.Filtered) - 1
	} else {
		m.List.Cursor = 0
	}

	// If cursor is at 0 (initial state), ensure YOffset is also 0
	if m.List.Cursor == 0 {
		m.List.Viewport.YOffset = 0
	}

	m.setRenderItem()

	// Note: We don't call SetContent here because the View() method uses
	// VisibleContent() to render only the visible portion. Calling SetContent
	// with View() would cause conflicting YOffset adjustments.
	if m.ready {
		l().Info("StacksView.setStacks: Content ready for rendering")
	} else {
		l().Warn("StacksView.setStacks: View not ready yet")
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

		cols := 2
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

		// Update cached widths so header stays aligned after resize
		m.width = width

		nameCol := fmt.Sprintf("%-*s", colWidths[0], s.Name)
		svcCol := fmt.Sprintf("%-*d", colWidths[1], s.ServiceCount)
		line := nameCol + svcCol

		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			nameCol = selStyle.Render(fmt.Sprintf("%-*s", colWidths[0], s.Name))
			svcCol = selStyle.Render(fmt.Sprintf("%-*d", colWidths[1], s.ServiceCount))
			return nameCol + svcCol
		}
		return itemStyle.Render(line)
	}
}

// GetStacksHelpContent returns categorized help for the stacks view
func GetStacksHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i/enter>", Description: "Show services for Stack"},
				{Keys: "<p>", Description: "Show tasks for Stack"},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+s>", Description: "Order by Stack name (todo)"},
				{Keys: "<shift+e>", Description: "Order by Services name (todo)"},
				{Keys: "<shift+t>", Description: "Order by Tasks name (todo)"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Navigate"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<q>", Description: "Quit"},
			},
		},
	}
}
