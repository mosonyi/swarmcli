package configsview

import (
	"context"
	"fmt"
	"strings"
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
	var contentLines []string
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

	// Compute frameHeight similarly to stacks view: reserve two lines from
	// the viewport height for surrounding UI and fall back to model height
	// minus reserved lines when viewport hasn't been initialized yet.
	frameHeight := m.configsList.Viewport.Height - 2
	if frameHeight <= 0 {
		if m.height > 0 {
			frameHeight = m.height - 4
		}
		if frameHeight <= 0 {
			frameHeight = 20
		}
	}

	// Header occupies one line when present
	headerLines := 0
	if header != "" {
		headerLines = 1
	}

	// Footer lines
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}

	// Desired content lines inside the box (not counting borders)
	desiredContentLines := frameHeight - 2 - headerLines - footerLines
	if desiredContentLines < 0 {
		desiredContentLines = 0
	}

	// Determine content before trimming/padding by computing desired lines
	// and asking the FilterableList to render that exact slice.
	content = m.configsList.VisibleContent(desiredContentLines)
	contentLines = strings.Split(content, "\n")
	// Trim trailing empty lines for stable calculation
	for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}

	// Pad or trim content to desired length
	if len(contentLines) < desiredContentLines {
		for i := 0; i < desiredContentLines-len(contentLines); i++ {
			contentLines = append(contentLines, "")
		}
	} else if len(contentLines) > desiredContentLines {
		contentLines = contentLines[:desiredContentLines]
	}

	paddedContent := strings.Join(contentLines, "\n")

	// Apply overlays to padded content BEFORE framing
	if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		paddedContent = ui.OverlayCentered(paddedContent, fileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		paddedContent = ui.OverlayCentered(paddedContent, createDialog, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		paddedContent = ui.OverlayCentered(paddedContent, dialogView, width, 0)
	} else if m.errorDialogActive {
		errorDialog := errordialog.Render(fmt.Sprintf("%v", m.err))
		paddedContent = ui.OverlayCentered(paddedContent, errorDialog, width, 0)
	} else if m.state == stateLoading || m.loadingView.Visible() {
		loadingView := m.loadingView.View()
		paddedContent = ui.OverlayCentered(paddedContent, loadingView, width, 0)
	}

	// Use viewport width for frame width (fallback to model width), add 4
	// to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.configsList.Viewport.Width
	if frameWidth <= 0 {
		frameWidth = m.width
	}
	frameWidth = frameWidth + 4

	title := fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
	view := ui.RenderFramedBoxHeight(title, header, paddedContent, footer, frameWidth, frameHeight)

	return view
}

func renderConfigsHeader(items []configItem, width int) string {
	if len(items) == 0 {
		return "NAME         ID                 CONFIG USED      CREATED AT             UPDATED AT"
	}
	// Compute proportional widths for 5 columns: NAME | ID | USED | CREATED | UPDATED
	if width <= 0 {
		width = 80
	}
	// In header context, the caller will have already determined viewport width.
	// We'll attempt to use the current terminal width via lipgloss if possible,
	// but fall back to 80 if not available. The parent view will set header
	// line into the frame width, so we just compute equal partitions.
	cols := 5
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

	labels := []string{" NAME", "ID", "CONFIG USED", "CREATED AT", "UPDATED AT"}
	return ui.RenderColumnHeader(labels, colWidths)
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
	content := m.usedByList.View()
	footer := m.renderUsedByFooter()

	// Pad content to fill viewport height
	height := m.usedByList.Viewport.Height
	if height <= 0 {
		if m.height > 0 {
			height = m.height - 2
		}
		if height <= 0 {
			height = 20
		}
	}
	contentLines := strings.Split(content, "\n")
	availableLines := height - 4
	if availableLines < 0 {
		availableLines = 0
	}
	for len(contentLines) < availableLines {
		contentLines = append(contentLines, "")
	}
	paddedContent := strings.Join(contentLines, "\n")

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.usedByList.Viewport.Width + 4

	title := fmt.Sprintf("Config: %s - Used By Stacks (%d)", m.usedByConfigName, len(m.usedByList.Filtered))
	frameHeight := height - 2
	if frameHeight < 0 {
		frameHeight = 0
	}
	return ui.RenderFramedBoxHeight(title, header, paddedContent, footer, frameWidth, frameHeight)
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
