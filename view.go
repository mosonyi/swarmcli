package main

import (
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

	helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.systemInfo.View(),
		m.nodes.View(),
		helpText,
	)
}
