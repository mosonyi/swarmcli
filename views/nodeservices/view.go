package nodeservicesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	height := m.viewport.Height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	// --- Main nodeservices content ---
	header := ui.FrameHeaderStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		m.serviceColWidth, "SERVICE",
		m.stackColWidth, "STACK",
		m.replicaColWidth, "REPLICAS",
	))
	baseContent := ui.RenderFramedBox(m.title, header, m.viewport.View(), width, false)

	// --- Overlay confirm dialog if visible ---
	if m.confirmDialog.Visible {
		dialogContent := m.confirmDialog.View()
		return overlayCentered(baseContent, dialogContent, width, height)
	}

	return baseContent
}

// overlayCentered safely overlays a small dialog on top of a base box
func overlayCentered(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	canvasHeight := len(baseLines)
	canvas := make([]string, canvasHeight)
	copy(canvas, baseLines)

	overlayLines := strings.Split(overlay, "\n")
	dialogHeight := len(overlayLines)
	if dialogHeight == 0 {
		return base
	}

	// Compute overlay width
	dialogWidth := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > dialogWidth {
			dialogWidth = w
		}
	}

	// Center vertically within the base box (skip top/bottom border)
	startRow := (canvasHeight - dialogHeight) / 2
	if startRow < 1 {
		startRow = 1
	}
	if startRow+dialogHeight > canvasHeight-1 {
		startRow = canvasHeight - dialogHeight - 1
		if startRow < 1 {
			startRow = 1
		}
	}

	// Center horizontally within the base box (preserve left/right border)
	for i, line := range overlayLines {
		row := startRow + i
		if row <= 0 || row >= canvasHeight-1 {
			continue
		}

		lineWidth := lipgloss.Width(line)
		leftPad := (width - 2 - lineWidth) / 2 // subtract 2 for left/right border
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := width - 2 - lineWidth - leftPad
		if rightPad < 0 {
			rightPad = 0
		}

		canvas[row] = "│" + strings.Repeat(" ", leftPad) + line + strings.Repeat(" ", rightPad) + "│"
	}

	return strings.Join(canvas, "\n")
}

func (m *Model) renderEntries() string {
	if len(m.entries) == 0 {
		return "No services found."
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	const minService = 15
	const minStack = 10
	const replicaWidth = 10
	const gap = 2

	available := width - replicaWidth - 2*gap
	serviceCol := available / 2
	stackCol := available - serviceCol

	if serviceCol < minService {
		serviceCol = minService
	}
	if stackCol < minStack {
		stackCol = minStack
	}

	m.serviceColWidth = serviceCol
	m.stackColWidth = stackCol
	m.replicaColWidth = replicaWidth

	var lines []string
	for i, e := range m.entries {
		replicas := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		switch {
		case e.ReplicasTotal == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("—")
		case e.ReplicasOnNode == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(replicas)
		case e.ReplicasOnNode < e.ReplicasTotal:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(replicas)
		default:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(replicas)
		}

		serviceName := truncateWithEllipsis(e.ServiceName, serviceCol)
		stackName := truncateWithEllipsis(e.StackName, stackCol)

		line := fmt.Sprintf(
			"%-*s  %-*s  %*s",
			serviceCol, serviceName,
			stackCol, stackName,
			replicaWidth, replicas,
		)

		if i == m.cursor {
			line = ui.CursorStyle.Render(line)
		}

		lines = append(lines, line)
	}

	status := fmt.Sprintf(" Service %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", ui.StatusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	if maxWidth == 2 {
		return s[:1] + "…"
	}
	return s[:maxWidth-1] + "…"
}
