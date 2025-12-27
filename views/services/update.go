package servicesview

import (
	"context"
	"fmt"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	"swarmcli/views/view"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		l().Infof("ServicesView: Received Msg with %d entries", len(msg.Entries))
		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg.Entries)
		if err != nil {
			l().Errorf("ServicesView: Error computing hash: %v", err)
			return nil
		}
		m.SetContent(msg)
		m.Visible = true
		m.List.Viewport.SetContent(m.List.View())
		// Continue polling
		return tickCmd()

	case TickMsg:
		l().Infof("ServicesView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes (this will return either a Msg or the next TickMsg)
		if m.Visible {
			return CheckServicesCmd(m.lastSnapshot, m.filterType, m.nodeID, m.stackName)
		}
		// Continue polling even if not visible
		return tickCmd()

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		m.List.Viewport.SetContent(m.List.View())
		return nil
	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			m.loading.SetVisible(true)
			m.loadingViewMessage(entry.ServiceName)
			l().Debugln("Starting restartServiceWithProgressCmd for", entry.ServiceName)

			// create new channel for this operation
			m.msgCh = make(chan tea.Msg)

			return tea.Batch(
				restartServiceWithProgressCmd(entry.ServiceName, m.msgCh),
				m.listenForMessages(),
			)
		}
		return nil

	case serviceProgressMsg:
		l().Debugf("[UI] Received progress: %d/%d\n", msg.Progress.Replaced, msg.Progress.Total)

		m.loadingViewMessage(fmt.Sprintf(
			"Progress: %d/%d tasks replaced...",
			msg.Progress.Replaced, msg.Progress.Total,
		))

		if msg.Progress.Replaced == msg.Progress.Total && msg.Progress.Total > 0 {
			l().Debugln("[UI] Restart finished")
			m.loading.SetVisible(false)
			return tea.Batch(
				refreshServicesCmd(m.nodeID, m.stackName, m.filterType),
			)
		}

		return m.listenForMessages()

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}

		if m.loading.Visible() {
			return nil
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		switch msg.String() {
		case "i":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					content, err := docker.Inspect(context.Background(), docker.InspectService, entry.ServiceID)
					if err != nil {
						content = fmt.Sprintf("Error inspecting service %q: %v", entry.ServiceName, err)
					}
					return view.NavigateToMsg{
						ViewName: inspectview.ViewName,
						Payload: map[string]interface{}{
							"title": fmt.Sprintf("Service: %s", entry.ServiceName),
							"json":  content,
						},
					}
				}
			}
		case "r":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				m.confirmDialog.Visible = true
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
			}
		case "l":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						Payload:  entry,
						ViewName: logsview.ViewName,
					}
				}
			}
		case "q":
			m.Visible = false
			// Go back to stacks view
			return func() tea.Msg { return view.NavigateToMsg{ViewName: "stacks", Payload: nil} }

		case "esc":
			// ESC should also go back to stacks view
			m.Visible = false
			return func() tea.Msg { return view.NavigateToMsg{ViewName: "stacks", Payload: nil} }
		}

		m.List.Viewport.SetContent(m.List.View())
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(msg Msg) {
	l().Infof("ServicesView.SetContent: Updating display with %d services", len(msg.Entries))

	m.title = msg.Title

	m.List.Items = msg.Entries
	m.List.ApplyFilter()

	m.filterType = msg.FilterType
	m.nodeID = msg.NodeID
	m.stackName = msg.StackName

	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		m.List.Viewport.GotoTop()
	}
}

func (m *Model) setRenderItem() {
	// We'll compute column widths based on the longest service name then
	// allocate the remaining space to the other columns.
	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		width := m.List.Viewport.Width
		if width <= 0 {
			width = 80
		}

		cols := 6
		sepLen := 2
		sepTotal := sepLen * (cols - 1)
		// Effective width available for columns (excluding separators)
		effWidth := width - sepTotal
		if effWidth < cols { // ensure sensible minimum
			effWidth = width
		}
		colWidths := make([]int, cols)

		// Headers and sensible minimums
		headers := []string{" SERVICE", "STACK", "REPLICAS", "STATUS", "CREATED", "UPDATED"}
		minCols := make([]int, cols)
		for i := 0; i < cols; i++ {
			hw := lipgloss.Width(headers[i])
			floor := 6
			switch i {
			case 0:
				floor = 10
			case 1:
				floor = 10
			case 2:
				floor = 8
			case 3:
				floor = 8
			case 4, 5:
				floor = 8
			}
			if hw > floor {
				minCols[i] = hw
			} else {
				minCols[i] = floor
			}
		}

		// Find longest service name visually among items (include header)
		maxSvc := lipgloss.Width(headers[0])
		for _, it := range m.List.Items {
			if s, ok := any(it).(docker.ServiceEntry); ok {
				if w := lipgloss.Width(s.ServiceName); w > maxSvc {
					maxSvc = w
				}
			}
		}
		desiredSvc := maxSvc + 1 // reserve 1 for leading space

		// Assign desired service width if possible, otherwise use proportional
		// partitioning but ensure minimums.
		nonServiceMinSum := 0
		for i := 1; i < cols; i++ {
			nonServiceMinSum += minCols[i]
		}

		if desiredSvc+nonServiceMinSum <= effWidth {
			colWidths[0] = desiredSvc
			// Start with minimums for remaining columns
			for i := 1; i < cols; i++ {
				colWidths[i] = minCols[i]
			}
			// Distribute leftover space equally among columns 1..5 (within effWidth)
			sum := 0
			for _, v := range colWidths {
				sum += v
			}
			leftover := effWidth - sum
			if leftover > 0 {
				per := leftover / (cols - 1)
				rem := leftover % (cols - 1)
				for i := 1; i < cols; i++ {
					add := per
					if rem > 0 {
						add++
						rem--
					}
					colWidths[i] += add
				}
			}
		} else {
			// Proportional partition but respect minimums
			// Partition across the effective width
			base := effWidth / cols
			for i := 0; i < cols; i++ {
				colWidths[i] = base
			}
			for i := 0; i < cols; i++ {
				if colWidths[i] < minCols[i] {
					colWidths[i] = minCols[i]
				}
			}
			// Adjust last column to ensure total equals effWidth
			sum := 0
			for _, v := range colWidths {
				sum += v
			}
			if sum != effWidth {
				colWidths[cols-1] += effWidth - sum
				if colWidths[cols-1] < 1 {
					colWidths[cols-1] = 1
				}
			}
		}

		// Cache for header alignment
		m.colServiceWidth = colWidths[0]
		m.colStackWidth = colWidths[1]

		// Prepare texts
		replicasText := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		if e.ReplicasTotal == 0 {
			replicasText = "—"
		}

		// Truncate columns except the first (we try not to shorten first column
		// but if it still doesn't fit due to small viewport, fall back to
		// truncation there too).
		lastIdx := len(colWidths) - 1
		svcTruncWidth := colWidths[0]
		// If we reserved a leading space in formatting, reduce by 1
		if svcTruncWidth > 0 {
			svcTruncWidth = svcTruncWidth - 1
		}
		serviceName := truncateWithEllipsis(e.ServiceName, svcTruncWidth)
		stackName := truncateWithEllipsis(e.StackName, colWidths[1]-1)
		statusText := truncateWithEllipsis(e.Status, colWidths[3]-1)
		created := truncateWithEllipsis(formatRelativeTime(e.CreatedAt), colWidths[4]-1)
		updated := truncateWithEllipsis(formatRelativeTime(e.UpdatedAt), colWidths[lastIdx])

		itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
		col0 := itemStyle.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName))
		col1 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[1]-1, stackName))

		var replicasColor lipgloss.Color
		switch {
		case e.ReplicasTotal == 0:
			replicasColor = lipgloss.Color("8")
		case e.ReplicasOnNode == 0:
			replicasColor = lipgloss.Color("9")
		case e.ReplicasOnNode < e.ReplicasTotal:
			replicasColor = lipgloss.Color("11")
		default:
			replicasColor = lipgloss.Color("10")
		}
		replicasStyle := lipgloss.NewStyle().Foreground(replicasColor)
		col2 := replicasStyle.Render(fmt.Sprintf("%-*s", colWidths[2]-1, replicasText))

		statusColor := getStatusColor(e.Status)
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		col3 := statusStyle.Render(fmt.Sprintf("%-*s", colWidths[3]-1, statusText))

		col4 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[4]-1, created))
		col5 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[5], updated))

		// Join with two-space separators for readability
		sep := strings.Repeat(" ", sepLen)
		line := col0 + sep + col1 + sep + col2 + sep + col3 + sep + col4 + sep + col5

		if selected {
			selBg := lipgloss.Color("63")
			selBase := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			selRep := lipgloss.NewStyle().Foreground(replicasColor).Background(selBg).Bold(true)
			selStatus := lipgloss.NewStyle().Foreground(statusColor).Background(selBg).Bold(true)

			// Render each styled column including the separator so the
			// highlight background is continuous across the whole line.
			sepStr := strings.Repeat(" ", sepLen)
			col0 = selBase.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName) + sepStr)
			col1 = selBase.Render(fmt.Sprintf("%-*s", colWidths[1]-1, stackName) + sepStr)
			col2 = selRep.Render(fmt.Sprintf("%-*s", colWidths[2]-1, replicasText) + sepStr)
			col3 = selStatus.Render(fmt.Sprintf("%-*s", colWidths[3]-1, statusText) + sepStr)
			col4 = selBase.Render(fmt.Sprintf("%-*s", colWidths[4]-1, created) + sepStr)
			col5 = selBase.Render(fmt.Sprintf("%-*s", colWidths[5], updated))
			line = col0 + col1 + col2 + col3 + col4 + col5
		}
		return line
	}
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	// A maxWidth of 1 should render as the ellipsis
	if maxWidth <= 1 {
		return "…"
	}

	// If the string already fits in the available visual width, return it.
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Reserve width for the ellipsis
	ell := "…"
	ellW := lipgloss.Width(ell)

	// Build up runes until adding the next would exceed maxWidth-ellW
	var outRunes []rune
	cur := ""
	for _, r := range s {
		cur += string(r)
		if lipgloss.Width(cur)+ellW > maxWidth {
			break
		}
		outRunes = append(outRunes, r)
	}
	if len(outRunes) == 0 {
		return ell
	}
	return string(outRunes) + ell
}

// formatRelativeTime formats a time as a relative duration (e.g., "2h ago", "3d ago")
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else if d < 30*24*time.Hour {
		weeks := int(d.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	} else if d < 365*24*time.Hour {
		months := int(d.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo ago", months)
	} else {
		years := int(d.Hours() / 24 / 365)
		return fmt.Sprintf("%dy ago", years)
	}
}

// getStatusColor returns the appropriate color for a service status
func getStatusColor(status string) lipgloss.Color {
	switch status {
	case "updating", "rolling back":
		return lipgloss.Color("11") // yellow
	case "updated", "active":
		return lipgloss.Color("10") // green
	case "paused", "rollback paused":
		return lipgloss.Color("8") // gray
	case "rolled back":
		return lipgloss.Color("9") // red
	default:
		return lipgloss.Color("15") // white
	}
}
