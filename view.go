package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
	"swarmcli/styles"
	"swarmcli/views/logs"
)

func (m model) View() string {
	if m.inspecting {
		header := fmt.Sprintf("Inspecting (%s)", m.mode)
		if m.inspectSearchMode {
			header += fmt.Sprintf(" - Search: %s", m.inspectSearchTerm)
		}
		return styles.BorderStyle.Render(
			fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back, / to search]", header, m.inspectViewport.View()),
		)
	}

	if m.view == logs.ViewName {
		return m.logs.View()
	}

	if m.view == "nodeStacks" {
		var b strings.Builder
		b.WriteString("Stacks on node:\n\n")
		for i, stack := range m.nodeStacks {
			cursor := "  "
			if i == m.stackCursor {
				cursor = "➜ "
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, stack))
		}
		b.WriteString("\n[press enter to inspect logs, q/esc to go back]")
		return b.String()
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
