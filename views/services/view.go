package servicesview

import (
	"fmt"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// Compute dynamic column widths (same as in setRenderItem)
	replicaWidth := 10
	maxService := len("SERVICE")
	maxStack := len("STACK")
	for _, e := range m.List.Filtered {
		if len(e.ServiceName) > maxService {
			maxService = len(e.ServiceName)
		}
		if len(e.StackName) > maxStack {
			maxStack = len(e.StackName)
		}
	}
	total := maxService + maxStack + replicaWidth + 4
	if total > width {
		overflow := total - width
		if maxStack > maxService {
			maxStack -= overflow
			if maxStack < 5 {
				maxStack = 5
			}
		} else {
			maxService -= overflow
			if maxService < 5 {
				maxService = 5
			}
		}
	}

	header := ui.FrameHeaderStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		maxService, "SERVICE",
		maxStack, "STACK",
		replicaWidth, "REPLICAS",
	))

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Node %d of %d", m.List.Cursor+1, len(m.List.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.List.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.List.Query)
	} else if m.List.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
	}

	if footer != "" {
		footer = statusBar + "\n" + footer
	} else {
		footer = statusBar
	}

	content := ui.RenderFramedBox(m.title, header, m.List.View(), footer, width)

	if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, m.List.Viewport.Height)
	}
	if m.loading.Visible() {
		content = ui.OverlayCentered(content, m.loading.View(), width, m.List.Viewport.Height)
	}

	return content
}
