package networksview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/ui/components/sorting"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	// If in UsedBy view, render it instead of the main networks view
	if m.usedByViewActive {
		return m.renderUsedByView()
	}

	// If in inspect view, render it
	if m.inspectViewActive {
		return m.renderInspectView()
	}

	width := 80
	if m.networksList.Viewport.Width > 0 {
		width = m.networksList.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	header := m.renderNetworksHeader(m.networksList.Items, width)
	footer := m.renderNetworksFooter()

	frame := ui.ComputeFrameDimensions(
		m.networksList.Viewport.Width,
		m.networksList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	content := m.networksList.VisibleContent(frame.DesiredContentLines)
	if m.state == stateLoading && len(m.networksList.Items) == 0 && m.networksList.Mode != filterlist.ModeSearching {
		// Keep the normal frame + header, but show a loading message in the list area.
		lines := frame.DesiredContentLines
		if lines < 1 {
			lines = 1
		}
		parts := make([]string, lines)
		parts[0] = "Loading..."
		for i := 1; i < lines; i++ {
			parts[i] = ""
		}
		content = strings.Join(parts, "\n")
	}

	// Apply overlays to content BEFORE framing
	if m.createDialogActive {
		dialogView := m.renderCreateNetworkDialog(width)
		content = ui.OverlayCentered(content, dialogView, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		content = ui.OverlayCentered(content, dialogView, width, 0)
	} else if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	}

	title := fmt.Sprintf("Docker Networks (%d)", len(m.networksList.Filtered))
	view := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	return view
}

func (m *Model) renderCreateNetworkDialog(width int) string {
	maxW := 72
	if width > 0 && width-8 < maxW {
		maxW = width - 8
	}
	if maxW < 40 {
		maxW = 40
	}

	if m.createDriverIndex < 0 {
		m.createDriverIndex = 0
	}
	if m.createDriverIndex >= len(networkDriverOptions) {
		m.createDriverIndex = len(networkDriverOptions) - 1
	}

	selectedDriver := networkDriverOptions[m.createDriverIndex]

	checkbox := func(v bool) string {
		if v {
			return "[x]"
		}
		return "[ ]"
	}

	focusMark := func(idx int) string {
		if m.createInputFocus == idx && m.createDialogStep == "basic" {
			return "›"
		}
		return " "
	}

	commandPreview := func() string {
		args := []string{"docker network create"}
		if selectedDriver != "" {
			args = append(args, "--driver "+selectedDriver)
		}
		if m.createEnableIPv6 {
			args = append(args, "--ipv6")
		}

		ipv4Subnet := strings.TrimSpace(m.createIPv4Subnet.Value())
		ipv4Gateway := strings.TrimSpace(m.createIPv4Gateway.Value())
		if ipv4Subnet != "" {
			args = append(args, "--subnet "+ipv4Subnet)
		}
		if ipv4Gateway != "" {
			args = append(args, "--gateway "+ipv4Gateway)
		}

		ipv6Subnet := strings.TrimSpace(m.createIPv6Subnet.Value())
		ipv6Gateway := strings.TrimSpace(m.createIPv6Gateway.Value())
		if ipv6Subnet != "" {
			args = append(args, "--subnet "+ipv6Subnet)
		}
		if ipv6Gateway != "" {
			args = append(args, "--gateway "+ipv6Gateway)
		}

		if m.createAttachable {
			args = append(args, "--attachable")
		}
		if m.createInternal {
			args = append(args, "--internal")
		}
		name := strings.TrimSpace(m.createNameInput.Value())
		if name == "" {
			name = "<name>"
		}
		args = append(args, name)
		return strings.Join(args, " ")
	}

	var body []string
	switch m.createDialogStep {
	case "creating":
		body = append(body, "Creating Network")
		body = append(body, "")
		body = append(body, "Command:")
		body = append(body, "  "+commandPreview())
		body = append(body, "")
		body = append(body, fmt.Sprintf("Creating… %s", ui.SpinnerCharAt(m.spinner)))
		body = append(body, "")
		body = append(body, "Esc: close")
	case "review":
		body = append(body, "Review")
		body = append(body, "")
		body = append(body, "Command:")
		body = append(body, "  "+commandPreview())
		body = append(body, "")
		body = append(body, "Enter: create   Esc: cancel")
	default:
		body = append(body, "Create Network")
		body = append(body, "")
		body = append(body, focusMark(0)+" "+m.createNameInput.View())
		body = append(body, focusMark(1)+" Driver: "+selectedDriver+"  (←/→ to change)")
		body = append(body, "")
		body = append(body, "IPv4 Network Configuration")
		body = append(body, focusMark(2)+" "+m.createIPv4Subnet.View())
		body = append(body, focusMark(3)+" "+m.createIPv4Gateway.View())
		body = append(body, "")
		body = append(body, "IPv6 Network Configuration")
		body = append(body, focusMark(4)+" Enable IPv6: "+checkbox(m.createEnableIPv6)+"  (space to toggle)")
		body = append(body, focusMark(5)+" "+m.createIPv6Subnet.View())
		body = append(body, focusMark(6)+" "+m.createIPv6Gateway.View())
		body = append(body, "")
		body = append(body, "Advanced")
		body = append(body, focusMark(7)+" Isolated network: "+checkbox(m.createInternal)+"  (space to toggle)")
		body = append(body, focusMark(8)+" Manual container attachment: "+checkbox(m.createAttachable)+"  (space to toggle)")
		if m.createDialogError != "" {
			body = append(body, "")
			body = append(body, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(m.createDialogError))
		}
		body = append(body, "")
		body = append(body, "Tab: next field   Enter: review   Esc: cancel")
	}

	content := strings.Join(body, "\n")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117")).
		Padding(1, 2).
		Width(maxW)

	return boxStyle.Render(content)
}

func (m *Model) renderNetworksHeader(items []networkItem, width int) string {
	if width <= 0 {
		width = 80
	}

	nameWidth, driverWidth, scopeWidth, usedWidth, idWidth := m.networkColWidths(width)

	// Store for rendering items (header alignment / other rendering)
	m.colNameWidth = nameWidth
	m.colDriverWidth = driverWidth
	m.colScopeWidth = scopeWidth

	labels := []string{" NAME", "DRIVER", "SCOPE", "USED", "ID"}

	// Add sort indicators
	arrow := func() string {
		if m.sortAscending {
			return sorting.SortArrow(sorting.Ascending)
		}
		return sorting.SortArrow(sorting.Descending)
	}

	if m.sortField == SortByName {
		labels[0] = fmt.Sprintf(" NAME %s", arrow())
	}
	if m.sortField == SortByDriver {
		labels[1] = fmt.Sprintf("DRIVER %s", arrow())
	}
	if m.sortField == SortByScope {
		labels[2] = fmt.Sprintf("SCOPE %s", arrow())
	}
	if m.sortField == SortByUsed {
		labels[3] = fmt.Sprintf("USED %s", arrow())
	}
	if m.sortField == SortByID {
		labels[4] = fmt.Sprintf("ID %s", arrow())
	}

	sep := strings.Repeat(" ", 2)
	line := fmt.Sprintf("%-*s%s%-*s%s%-*s%s%-*s%s%-*s",
		nameWidth, labels[0],
		sep,
		driverWidth, labels[1],
		sep,
		scopeWidth, labels[2],
		sep,
		usedWidth, labels[3],
		sep,
		idWidth, labels[4],
	)

	return ui.FrameHeaderStyle.Render(line)
}

func (m *Model) renderNetworksFooter() string {
	if m.toastMessage != "" && time.Now().Before(m.toastUntil) {
		return ui.StatusBarStyle.Render(m.toastMessage)
	}
	if m.toastMessage != "" && time.Now().After(m.toastUntil) {
		m.toastMessage = ""
	}

	if m.state == stateLoading {
		return ui.StatusBarStyle.Render(fmt.Sprintf("Loading Docker networks… %s", ui.SpinnerCharAt(m.spinner)))
	}

	if m.networksList.Mode == filterlist.ModeSearching {
		prompt := fmt.Sprintf("Filter: %s", m.networksList.Query)
		return ui.StatusBarStyle.Render(prompt)
	}

	total := len(m.networksList.Items)
	filtered := len(m.networksList.Filtered)
	cursor := m.networksList.Cursor + 1

	if filtered == 0 {
		return ui.StatusBarStyle.Render("No networks")
	}

	if m.HasActiveFilter() {
		return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d (filtered from %d)", cursor, filtered, total))
	}

	return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d", cursor, total))
}

func (m *Model) renderUsedByView() string {
	width := 80
	if m.usedByList.Viewport.Width > 0 {
		width = m.usedByList.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	header := m.renderUsedByHeader(width)
	footer := m.renderUsedByFooter()

	frame := ui.ComputeFrameDimensions(
		m.usedByList.Viewport.Width,
		m.usedByList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	content := m.usedByList.VisibleContent(frame.DesiredContentLines)

	// Apply overlays
	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	}

	title := fmt.Sprintf("Network '%s' - Used By (%d)", m.usedByNetworkName, len(m.usedByList.Filtered))
	view := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	return view
}

func (m *Model) renderUsedByHeader(width int) string {
	return ui.RenderColumnHeader([]string{" STACK", "SERVICE"}, []int{32, 50})
}

func (m *Model) renderUsedByFooter() string {
	if m.usedByList.Mode == filterlist.ModeSearching {
		prompt := fmt.Sprintf("Filter: %s", m.usedByList.Query)
		return ui.StatusBarStyle.Render(prompt)
	}

	total := len(m.usedByList.Items)
	filtered := len(m.usedByList.Filtered)
	cursor := m.usedByList.Cursor + 1

	if filtered == 0 {
		return ui.StatusBarStyle.Render("No services using this network")
	}

	if len(m.usedByList.Query) > 0 {
		return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d (filtered from %d)", cursor, filtered, total))
	}

	return ui.StatusBarStyle.Render(fmt.Sprintf("%d/%d", cursor, total))
}

func (m *Model) renderInspectView() string {
	width := m.networksList.Viewport.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}

	header := ui.FrameHeaderStyle.Render("Network Details (JSON)")
	footerText := "↑/↓ Scroll | PgUp/PgDn Page | / Search | Esc Back"
	if m.inspectSearchMode {
		footerText = fmt.Sprintf("Search: %s  (Enter apply | Esc cancel)", m.inspectSearchTerm)
	}
	footer := ui.StatusBarStyle.Render(footerText)

	frame := ui.ComputeFrameDimensions(
		width,
		m.networksList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	content := ui.TrimOrPadContentToLines(m.inspectViewport.View(), frame.DesiredContentLines)

	// Apply overlays
	if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	}

	title := "Inspect Network"
	view := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	return view
}
