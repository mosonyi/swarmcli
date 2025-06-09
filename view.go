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
		return m.stacks.View()
	}

	// Todo: Extract this into a separate view
	status := styles.StatusStyle.Render(fmt.Sprintf(
		"Host: %s\nVersion: %s\nCPU: %s\nMEM: %s\nContainers: %d\nServices: %d",
		m.host, m.version, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	))

	helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		m.nodesV.View(),
		helpText,
	)
}
