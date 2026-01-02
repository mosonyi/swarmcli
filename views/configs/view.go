package configsview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	filterlist "swarmcli/ui/components/filterable/list"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

// Shared dialog styles
var (
	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("117"))

	dialogItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	dialogSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	dialogKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)
)

type configItem struct {
	Name      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Used      bool // true if used by any service
	UsedKnown bool // true if Used has been computed (false => loading/unknown)
}

func (i configItem) FilterValue() string { return i.Name }
func (i configItem) Title() string       { return i.Name }
func (i configItem) Description() string {
	createdStr := "N/A"
	if !i.CreatedAt.IsZero() {
		createdStr = i.CreatedAt.Format("2006-01-02 15:04:05")
	}
	updatedStr := "N/A"
	if !i.UpdatedAt.IsZero() {
		updatedStr = i.UpdatedAt.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("ID: %s        Created: %s        Updated: %s", i.ID, createdStr, updatedStr)
}

type usedByItem struct {
	StackName   string
	ServiceName string
}

func (i usedByItem) FilterValue() string { return i.StackName + " " + i.ServiceName }
func (i usedByItem) Title() string       { return fmt.Sprintf("%-24s %-24s", i.StackName, i.ServiceName) }
func (i usedByItem) Description() string { return "Service: " + i.ServiceName }

func configItemFromSwarm(ctx context.Context, c swarm.Config) configItem {
	used := false
	services, err := docker.ListServicesUsingConfigID(ctx, c.ID)
	if err == nil && len(services) > 0 {
		used = true
	}
	return configItem{
		Name:      c.Spec.Name,
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Used:      used,
		UsedKnown: true,
	}
}

func (m *Model) View() string {
	// If in UsedBy view, render it instead of the main configs view
	if m.usedByViewActive {
		return m.renderUsedByView()
	}

	width := 80
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	} else if m.width > 0 {
		width = m.width
	}

	header := renderConfigsHeader(m.configsList.Items, width)

	// Fixme: https://github.com/mosonyi/swarmcli/issues/141
	nameCol := m.colNameWidth
	idCol := m.colIdWidth
	if nameCol <= 0 {
		nameCol = len("NAME")
	}
	if idCol <= 0 {
		idCol = len("ID")
	}
	// column width calculations are handled in setRenderItem; the
	// FilterableList's RenderItem now controls formatting.
	for _, cfg := range m.configsList.Items {
		if len(cfg.Name) > nameCol {
			nameCol = len(cfg.Name)
		}
		if len(cfg.ID) > idCol {
			idCol = len(cfg.ID)
		}
	}
	// Render exactly desiredContentLines rows from the configs list without
	// mutating the viewport height each render to prevent jitter.
	// We'll compute desiredContentLines below and then call VisibleContent.
	// (placeholder for content variable; actual value assigned later)
	var content string
	footer := m.renderConfigsFooter()

	frame := ui.ComputeFrameDimensions(
		m.configsList.Viewport.Width,
		m.configsList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	// Use VisibleContent to get only the visible portion based on cursor position
	// This ensures proper scrolling and that the cursor is always visible
	// VisibleContent already returns exactly desiredContentLines, so we use
	// RenderFramedBox instead of RenderFramedBoxHeight to avoid double-padding
	content = m.configsList.VisibleContent(frame.DesiredContentLines)

	// Apply overlays to content BEFORE framing
	if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		content = ui.OverlayCentered(content, fileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		content = ui.OverlayCentered(content, createDialog, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		content = ui.OverlayCentered(content, dialogView, width, 0)
	} else if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	} else if m.state == stateLoading || m.loadingView.Visible() {
		loadingView := m.loadingView.View()
		content = ui.OverlayCentered(content, loadingView, width, 0)
	}

	title := fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
	view := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	return view
}

func renderConfigsHeader(items []configItem, width int) string {
	if len(items) == 0 {
		return "NAME         ID                 CONFIG USED      CREATED AT             UPDATED AT"
	}
	// Compute proportional widths using the same logic as the row renderer
	if width <= 0 {
		width = 80
	}
	cols := 5
	sepLen := 2
	starts := make([]int, cols)
	for i := 0; i < cols; i++ {
		starts[i] = (i * width) / cols
	}
	colWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i == cols-1 {
			colWidths[i] = width - starts[i]
		} else {
			colWidths[i] = starts[i+1] - starts[i]
		}
		if colWidths[i] < 1 {
			colWidths[i] = 1
		}
	}

	// Ensure CREATED and UPDATED columns have at least 19 chars
	minTime := 19
	cur := colWidths[3] + colWidths[4]
	if cur < 2*minTime {
		deficit := 2*minTime - cur
		for i := 1; i >= 0 && deficit > 0; i-- {
			take := deficit
			if colWidths[i] > take+5 {
				colWidths[i] -= take
				deficit = 0
			} else {
				take = colWidths[i] - 5
				if take > 0 {
					colWidths[i] -= take
					deficit -= take
				}
			}
		}
		if colWidths[3] < minTime {
			colWidths[3] = minTime
		}
		if colWidths[4] < minTime {
			colWidths[4] = minTime
		}
	}

	if colWidths[2] < 1 {
		colWidths[2] = 1
	}

	// Add separators back for header rendering
	headerRenderWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i < cols-1 {
			headerRenderWidths[i] = colWidths[i] + sepLen
		} else {
			headerRenderWidths[i] = colWidths[i]
		}
	}
	labels := []string{" NAME", "ID", "CONFIG USED", "CREATED AT", "UPDATED AT"}
	return ui.RenderColumnHeader(labels, headerRenderWidths)
}
func (m *Model) renderConfigsFooter() string {
	status := fmt.Sprintf("Config %d of %d", m.configsList.Cursor+1, len(m.configsList.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.configsList.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.configsList.Query)
	} else if m.configsList.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.configsList.Query)
	}

	if footer != "" {
		return statusBar + "\n" + footer
	}
	return statusBar
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialogTitleStyle.Render(" Create Config - Choose Source "))
		lines = append(lines, dialogItemStyle.Render(""))
		lines = append(lines, dialogItemStyle.Render("How would you like to create the config?"))
		lines = append(lines, dialogItemStyle.Render(""))

		if m.createConfigSource == "file" {
			lines = append(lines, dialogSelectedStyle.Render("→ From file"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  From file"))
		}

		if m.createConfigSource == "inline" {
			lines = append(lines, dialogSelectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  Inline editor"))
		}

		lines = append(lines, dialogItemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<↑>"),
			dialogKeyStyle.Render("<↓>"),
			dialogKeyStyle.Render("<Esc>"))
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, dialogTitleStyle.Render(" Create Config from File "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show file path input with browse indicator when focused
		fileLine := m.createFileInput.View()
		if m.createInputFocus == 1 {
			fileLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		}
		lines = append(lines, dialogItemStyle.Render(fileLine))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialogTitleStyle.Render(" Create Config - Inline Editor "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show editor status with edit hint when focused
		editorStatus := "Content: "
		if m.createConfigData != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createConfigData))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + dialogKeyStyle.Render("[e: Edit]")
		}
		lines = append(lines, dialogItemStyle.Render(editorStatus))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
}

func (m *Model) renderUsedByView() string {
	// Safety check - if list is not properly initialized, show error
	if m.usedByList.Viewport.Width == 0 {
		m.usedByViewActive = false
		return "Error: UsedBy view not properly initialized"
	}

	header := m.renderUsedByHeader()
	footer := m.renderUsedByFooter()

	// Compute frame dimensions to get the exact number of content lines needed
	frame := ui.ComputeFrameDimensions(
		m.usedByList.Viewport.Width,
		m.usedByList.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	// Use VisibleContent to get only the visible portion based on cursor position
	// This ensures proper scrolling and that the cursor is always visible
	content := m.usedByList.VisibleContent(frame.DesiredContentLines)

	title := fmt.Sprintf("Config: %s - Used By Stacks (%d)", m.usedByConfigName, len(m.usedByList.Filtered))
	return ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)
}

func (m *Model) renderUsedByHeader() string {
	width := m.usedByList.Viewport.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}
	cols := 2
	starts := make([]int, cols)
	for i := 0; i < cols; i++ {
		starts[i] = (i * width) / cols
	}
	colWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i == cols-1 {
			colWidths[i] = width - starts[i]
		} else {
			colWidths[i] = starts[i+1] - starts[i]
		}
		if colWidths[i] < 1 {
			colWidths[i] = 1
		}
	}

	// Uppercase labels and prefix first label with a leading space to align
	labels := []string{" STACK NAME", "SERVICE NAME"}
	return ui.RenderColumnHeader(labels, colWidths)
}

func (m *Model) renderUsedByFooter() string {
	totalStacks := len(m.usedByList.Items)
	if totalStacks == 0 {
		return ui.StatusBarStyle.Render("No stacks use this config")
	}

	status := fmt.Sprintf("Stack %d of %d", m.usedByList.Cursor+1, len(m.usedByList.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.usedByList.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.usedByList.Query)
	} else if m.usedByList.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.usedByList.Query)
	}

	if footer != "" {
		return statusBar + "\n" + footer
	}
	return statusBar
}
