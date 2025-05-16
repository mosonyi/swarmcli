package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m model) View() string {
	if m.inspecting {
		header := fmt.Sprintf("Inspecting (%s)", m.mode)
		if m.inspectSearchMode {
			header += fmt.Sprintf(" - Search: %s", m.inspectSearchTerm)
		}
		return borderStyle.Render(
			fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back, / to search]", header, m.inspectViewport.View()),
		)
	}

	if m.viewingLogs {
		header := fmt.Sprintf("Logs (%s)", m.mode)
		if m.stackLogsSearchMode {
			header += fmt.Sprintf(" - Search: %s", m.stackLogsSearchTerm)
		}
		return borderStyle.Render(
			fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back, / to search]", header, m.logsViewport.View()),
		)
	}
	//
	//if m.viewingLogs {
	//	return frame("Logs", m.logsViewport.View(), m.logsViewport.Width)
	//}

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

	status := statusStyle.Render(fmt.Sprintf(
		"Host: %s\nVersion: %s\nCPU: %s\nMEM: %s\nContainers: %d\nServices: %d",
		m.host, m.version, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	))

	helpText := helpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

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
		borderStyle.Render(s),
		helpText,
	)
}
