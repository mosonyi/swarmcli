package main

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if !m.initialized || m.currentView == nil {
		return "Initializing..."
	}
	//helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.systemInfo.View(),
		m.currentView.View(),
		m.renderStackBar(),
	)
}
