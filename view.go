package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/styles"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
	stacksview "swarmcli/views/stacks"
)

func (m model) View() string {
	if m.view == inspectview.ViewName {
		return m.inspect.View()
	}

	if m.view == logs.ViewName {
		return m.logs.View()
	}

	if m.view == stacksview.ViewName {
		return m.logs.View()
	}

	status := styles.StatusStyle.Render(fmt.Sprintf(
		"Host: %s\nVersion: %s\nCPU: %s\nMEM: %s\nContainers: %d\nServices: %d",
		m.host, m.version, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	))

	helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	// Show the main list with cursor highlighted, no viewport scroll for this version
	s := fmt.Sprintf("Mode: %s\n\n", m.mode)
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "→ "
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		styles.BorderStyle.Render(s),
		helpText,
	)
}
