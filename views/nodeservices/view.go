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
	if width <= 0 {
		width = 80
	}

	headerStyle := ui.FrameHeaderStyle
	header := headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		m.serviceColWidth, "SERVICE",
		m.stackColWidth, "STACK",
		m.replicaColWidth, "REPLICAS",
	))

	return ui.RenderFramedBox(m.title, header, m.viewport.View(), width)
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
