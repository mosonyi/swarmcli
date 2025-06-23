package main

import (
	"github.com/charmbracelet/lipgloss"
	"swarmcli/styles"
)

func (m model) View() string {
	//if m.view == inspectview.ViewName {
	//	return m.inspect.View()
	//}
	//
	//if m.view == logsview.ViewName {
	//	return m.logs.View()
	//}
	//
	//if m.view == stacksview.ViewName {
	//	return m.stacks.View()
	//}

	helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.systemInfo.View(),
		// nodesview,
		m.currentView.View(),
		helpText,
	)
}
