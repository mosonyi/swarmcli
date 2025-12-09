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
	StackName string
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
	}

	header := renderConfigsHeader(m.configsList.Items)
	var contentLines []string
	nameCol := len("NAME")
	idCol := len("ID")
	usedCol := len("CONFIG USED")
	space := 6 // extra space between columns
	for _, cfg := range m.configsList.Items {
		if len(cfg.Name) > nameCol {
			nameCol = len(cfg.Name)
		}
		if len(cfg.ID) > idCol {
			idCol = len(cfg.ID)
		}
	}
	for idx, item := range m.configsList.Items {
		usedStr := " "
		if item.Used {
			usedStr = "●"
		}
		row := fmt.Sprintf("%-*s%*s%-*s%*s%*s%*s%-19s%*s%-19s",
			nameCol, item.Name, space, "",
			idCol, item.ID, space, "",
			usedCol, usedStr, space, "",
			item.CreatedAt.Format("2006-01-02 15:04:05"), space, "",
			item.UpdatedAt.Format("2006-01-02 15:04:05"))
		if idx == m.configsList.Cursor {
			row = lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Background(lipgloss.Color("63")).Bold(true).Render(row)
		}
		contentLines = append(contentLines, row)
	}
	content := strings.Join(contentLines, "\n")
	footer := m.renderConfigsFooter()

	// Pad content to fill viewport height
	height := m.configsList.Viewport.Height
	if height <= 0 {
		height = 20
	}
	contentLines = strings.Split(content, "\n")
	// Account for frame borders (2), title (1), header (1) = 4 lines overhead
	availableLines := height - 4
	if availableLines < 0 {
		availableLines = 0
	}
	for len(contentLines) < availableLines {
		contentLines = append(contentLines, "")
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

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := width + 4

	title := fmt.Sprintf("Docker Configs (%d)", len(m.configsList.Filtered))
	view := ui.RenderFramedBox(title, header, paddedContent, footer, frameWidth)

	return view
}

func renderConfigsHeader(items []configItem) string {
    if len(items) == 0 {
        return "NAME         ID                 CONFIG USED      CREATED AT             UPDATED AT"
    }

    // Compute max widths
    nameCol := len("NAME")
    idCol := len("ID")
    usedCol := len("CONFIG USED")
    space := 6 // extra space between columns
    for _, cfg := range items {
        if len(cfg.Name) > nameCol {
            nameCol = len(cfg.Name)
        }
        if len(cfg.ID) > idCol {
            idCol = len(cfg.ID)
        }
    }
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("15")). // white
        Bold(true)
    return headerStyle.Render(fmt.Sprintf("%-*s%*s%-*s%*s%-*s%*s%-19s%*s%-19s", nameCol, "NAME", space, "", idCol, "ID", space, "", usedCol, "CONFIG USED", space, "", "CREATED AT", space, "", "UPDATED AT"))
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
		height = 20
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
	return ui.RenderFramedBox(title, header, paddedContent, footer, frameWidth)
}

func (m *Model) renderUsedByHeader() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	// Use fixed column widths for alignment
	return headerStyle.Render(fmt.Sprintf("%-24s %-24s", "Stack Name", "Service Name"))
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
