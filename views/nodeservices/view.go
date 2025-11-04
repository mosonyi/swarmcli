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

	// Show loading view if active
	if m.loading.Visible() {
		return m.loading.View()
	}

	// --- Render the main nodeservices content ---
	headerStyle := ui.FrameHeaderStyle
	header := headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		m.serviceColWidth, "SERVICE",
		m.stackColWidth, "STACK",
		m.replicaColWidth, "REPLICAS",
	))
	content := ui.RenderFramedBox(m.title, header, m.viewport.View(), width)

	// --- Overlay confirm dialog if visible ---
	if m.confirmDialog.Visible {
		dialogContent := m.confirmDialog.View()
		return overlayCentered(content, dialogContent, width, height)
	}

	return content
}

// Helpers
func splitLines(s string) []string {
	return strings.Split(s, "\n")
}

func padRight(s string, width int) string {
	l := lipgloss.Width(s)
	if l >= width {
		return s
	}
	return s + strings.Repeat(" ", width-l)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func overlayCentered(base, overlay string, width, height int) string {
	baseLines := splitLines(base)
	canvasHeight := max(len(baseLines), height)
	canvas := make([]string, canvasHeight)

	// Copy base lines safely
	for i := 0; i < canvasHeight; i++ {
		if i < len(baseLines) {
			canvas[i] = padRight(baseLines[i], width)
		} else {
			canvas[i] = strings.Repeat(" ", width)
		}
	}

	overlayLines := splitLines(overlay)
	dialogHeight := len(overlayLines)
	if dialogHeight == 0 {
		return strings.Join(canvas, "\n")
	}

	// Compute overlay width
	dialogWidth := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > dialogWidth {
			dialogWidth = w
		}
	}

	// Centering coordinates
	startRow := (canvasHeight - dialogHeight) / 2
	if startRow < 1 {
		startRow = 1 // leave top border
	}
	if startRow+dialogHeight > canvasHeight-1 {
		startRow = canvasHeight - dialogHeight - 1 // leave bottom border
		if startRow < 1 {
			startRow = 1
		}
	}

	startCol := (width - dialogWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Overlay the dialog safely
	for i, line := range overlayLines {
		row := startRow + i
		if row >= len(canvas)-1 || row < 0 { // never touch top/bottom border, never negative
			continue
		}
		padding := strings.Repeat(" ", startCol)
		rest := ""
		if startCol+lipgloss.Width(line) < width {
			rest = strings.Repeat(" ", width-startCol-lipgloss.Width(line))
		}
		canvas[row] = padding + line + rest
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
