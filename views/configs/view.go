package configsview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"
	filterlist "swarmcli/ui/components/filterable/list"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name      string
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
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

func configItemFromSwarm(c swarm.Config) configItem {
	return configItem{
		Name:      c.Spec.Name,
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func (m *Model) View() string {
	width := 80
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	}

	header := renderConfigsHeader(m.configsList.Items)
	content := m.configsList.View()
	footer := m.renderConfigsFooter()

	// Pad content to fill viewport height
	height := m.configsList.Viewport.Height
	if height <= 0 {
		height = 20
	}
	contentLines := strings.Split(content, "\n")
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
		return "NAME       ID               CREATED AT           UPDATED AT"
	}

	// Compute max widths
	nameCol := len("NAME")
	idCol := len("ID")
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
	return headerStyle.Render(fmt.Sprintf("%-*s        %-*s        %-19s        %-19s", nameCol, "NAME", idCol, "ID", "CREATED AT", "UPDATED AT"))
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
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117"))

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, titleStyle.Render(" Create Config - Choose Source "))
		lines = append(lines, itemStyle.Render(""))
		lines = append(lines, itemStyle.Render("How would you like to create the config?"))
		lines = append(lines, itemStyle.Render(""))

		if m.createConfigSource == "file" {
			lines = append(lines, selectedStyle.Render("→ From file"))
		} else {
			lines = append(lines, itemStyle.Render("  From file"))
		}

		if m.createConfigSource == "inline" {
			lines = append(lines, selectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, itemStyle.Render("  Inline editor"))
		}

		lines = append(lines, itemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			keyStyle.Render("<Enter>"),
			keyStyle.Render("<↑>"),
			keyStyle.Render("<↓>"),
			keyStyle.Render("<Esc>"))
		lines = append(lines, helpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, titleStyle.Render(" Create Config from File "))
		lines = append(lines, itemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, itemStyle.Render(""))
		}

		lines = append(lines, itemStyle.Render(m.createNameInput.View()))
		lines = append(lines, itemStyle.Render(""))

		// Show file path input with browse indicator when focused
		fileLine := m.createFileInput.View()
		if m.createInputFocus == 1 {
			fileLine += "  " + keyStyle.Render("[f: Browse]")
		}
		lines = append(lines, itemStyle.Render(fileLine))
		lines = append(lines, itemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				keyStyle.Render("<Enter>"),
				keyStyle.Render("<Tab>"),
				keyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				keyStyle.Render("<Enter>"),
				keyStyle.Render("<Tab>"),
				keyStyle.Render("<Esc>"))
		}
		lines = append(lines, helpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, titleStyle.Render(" Create Config - Inline Editor "))
		lines = append(lines, itemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1)
			lines = append(lines, errorStyle.Render("⚠ "+m.createDialogError))
			lines = append(lines, itemStyle.Render(""))
		}

		lines = append(lines, itemStyle.Render(m.createNameInput.View()))
		lines = append(lines, itemStyle.Render(""))

		// Show editor status with edit hint when focused
		editorStatus := "Content: "
		if m.createConfigData != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createConfigData))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + keyStyle.Render("[e: Edit]")
		}
		lines = append(lines, itemStyle.Render(editorStatus))
		lines = append(lines, itemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				keyStyle.Render("<Enter>"),
				keyStyle.Render("<Tab>"),
				keyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Confirm • %s Navigate • %s Cancel",
				keyStyle.Render("<Enter>"),
				keyStyle.Render("<Tab>"),
				keyStyle.Render("<Esc>"))
		}
		lines = append(lines, helpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}
