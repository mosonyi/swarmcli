// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/ui/components/sorting"
	"github.com/Eldara-Tech/swarmcli/views/taskutil"
	"strings"
)

func (m *Model) FrameTitle() string {
	return fmt.Sprintf("Tasks - Stack: %s (Total: %d)", m.stackName, len(m.tasks))
}

func (m *Model) FrameHeader() string { return "" }

func (m *Model) FrameFooter() string {
	return ui.StatusBarStyle.Render(fmt.Sprintf("Viewing %d tasks", len(m.tasks)))
}

func (m *Model) FrameContent() string {
	return m.viewport.View()
}

func (m *Model) View() string {
	if !m.visible {
		return ""
	}
	return ui.RenderFramedBox(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(), m.width)
}

func (m *Model) renderTasks() string {
	if len(m.tasks) == 0 {
		return "No tasks found for this stack."
	}

	width := m.viewport.Width
	if width < 80 {
		width = 80
	}

	// Column headers with percentage-based widths and spacing
	// Calculate widths: ID=14%, NAME=28%, IMAGE=20%, NODE=14%, STATE=12%, STATUS=remaining
	colWidths := []int{
		(width * 14) / 100, // ID
		(width * 28) / 100, // NAME
		(width * 20) / 100, // IMAGE
		(width * 14) / 100, // NODE
		(width * 12) / 100, // STATE
		0,                  // STATUS (calculated below)
	}
	colWidths[5] = width - colWidths[0] - colWidths[1] - colWidths[2] - colWidths[3] - colWidths[4]

	headerLabels := []string{"  ID", "NAME", "IMAGE", "NODE", "STATE", "STATUS"}

	// Add sort indicators to labels
	if m.sortField == SortByName {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		headerLabels[1] = fmt.Sprintf("NAME %s", arrow)
	}
	if m.sortField == SortByService {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		headerLabels[0] = fmt.Sprintf("  ID %s", arrow)
	}
	if m.sortField == SortByNode {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		headerLabels[3] = fmt.Sprintf("NODE %s", arrow)
	}
	if m.sortField == SortByState {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		headerLabels[4] = fmt.Sprintf("STATE %s", arrow)
	}

	headerLine := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s",
		colWidths[0], headerLabels[0],
		colWidths[1], headerLabels[1],
		colWidths[2], headerLabels[2],
		colWidths[3], headerLabels[3],
		colWidths[4], headerLabels[4],
		colWidths[5], headerLabels[5])
	header := ui.FrameHeaderStyle.Render(headerLine)

	var lines []string
	lines = append(lines, header)

	for i, task := range m.tasks {
		namePrefix := ""
		if i > 0 && task.ServiceName == m.tasks[i-1].ServiceName {
			namePrefix = " \\_ "
		}

		// Truncate values to fit column widths (accounting for leading spaces in first column)
		id := "  " + truncate(task.ID, colWidths[0]-2)
		name := truncate(namePrefix+task.Name, colWidths[1])
		image := truncate(task.Image, colWidths[2])
		node := truncate(task.NodeName, colWidths[3])
		desiredState := truncate(task.DesiredState, colWidths[4])

		status := task.StatusText()
		if task.Error != "" {
			status = "Failed: " + task.Error
		}
		status = truncate(status, colWidths[5])

		line := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s%-*s",
			colWidths[0], id,
			colWidths[1], name,
			colWidths[2], image,
			colWidths[3], node,
			colWidths[4], desiredState,
			colWidths[5], status)

		lines = append(lines, taskutil.TaskRowStyle(task, ui.ListItemStyle).Render(line))
	}

	return strings.Join(lines, "\n")
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen > 3 {
		return s[:maxLen-1] + "…"
	}
	return s[:maxLen]
}
