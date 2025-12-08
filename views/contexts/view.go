package contexts

import (
	"fmt"
	"path/filepath"
	"strings"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	title := "Docker Contexts"
	header := "Select a context to switch"

	if m.IsLoading() {
		header = "Loading contexts..."
	} else if m.IsSwitchPending() {
		header = "Switching context..."
	} else if err := m.GetError(); err != "" {
		header = fmt.Sprintf("Error: %s", err)
	} else if msg := m.GetSuccess(); msg != "" {
		header = msg
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// Build content
	var content strings.Builder
	contexts := m.GetContexts()
	cursor := m.GetCursor()

	if len(contexts) == 0 && !m.IsLoading() {
		content.WriteString("No Docker contexts found")
	} else {
		// Table header
		headerLine := fmt.Sprintf("%-4s %-20s %-4s %-60s %s", "CURR", "NAME", "TLS", "DESCRIPTION", "DOCKER ENDPOINT")
		content.WriteString(lipgloss.NewStyle().Bold(true).Render(headerLine))
		content.WriteString("\n")

		// Contexts list
		for i, ctx := range contexts {
			current := " "
			if ctx.Current {
				current = "*"
			}

			// Truncate long values
			name := ctx.Name
			if len(name) > 18 {
				name = name[:15] + "..."
			}

			// TLS indicator - show circle only if TLS is enabled
			tlsChar := " "
			if ctx.TLS {
				tlsChar = "●"
			}

			desc := ctx.Description
			if len(desc) > 58 {
				desc = desc[:55] + "..."
			}

			host := ctx.DockerHost
			if len(host) > 40 {
				host = host[:37] + "..."
			}

			// Build line
			line := fmt.Sprintf("%-4s %-20s %-4s %-60s %s", current, name, tlsChar, desc, host)

			// Highlight selected row
			if i == cursor {
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("230")).
					Render(line)
			}

			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	frameWidth := width + 4
	height := m.viewport.Height

	// Pad content to fill viewport height
	contentLines := strings.Split(content.String(), "\n")
	// Account for frame borders (2), title (1), header (1) = 4 lines overhead
	availableLines := height - 4
	if availableLines < 0 {
		availableLines = 0
	}
	for len(contentLines) < availableLines {
		contentLines = append(contentLines, "")
	}
	paddedContent := strings.Join(contentLines, "\n")

	// Overlay dialogs on content BEFORE framing
	if m.certFileBrowserActive {
		// Cert file browser has highest priority when open
		certFileBrowserDialog := m.renderCertFileBrowserDialog()
		paddedContent = ui.OverlayCentered(paddedContent, certFileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		paddedContent = ui.OverlayCentered(paddedContent, createDialog, width, 0)
	} else if m.editDialogActive {
		editDialog := m.renderEditDialog()
		paddedContent = ui.OverlayCentered(paddedContent, editDialog, width, 0)
	} else if m.errorDialogActive {
		errorDialog := m.renderErrorDialog()
		paddedContent = ui.OverlayCentered(paddedContent, errorDialog, width, 0)
	} else if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select .tar file", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		paddedContent = ui.OverlayCentered(paddedContent, fileBrowserDialog, width, 0)
	} else if m.importInputActive {
		importDialog := m.renderImportDialog()
		paddedContent = ui.OverlayCentered(paddedContent, importDialog, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		paddedContent = ui.OverlayCentered(paddedContent, dialogView, width, 0)
	}

	rendered := ui.RenderFramedBox(
		title,
		headerRendered,
		paddedContent,
		"",
		frameWidth,
	)

	return rendered
}

func (m *Model) renderImportDialog() string {
	contentWidth := 60

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1).
		Width(contentWidth)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(contentWidth)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117")).
		Width(contentWidth + 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(contentWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Import Docker Context "))
	lines = append(lines, itemStyle.Render("Enter the path to the context tar file:"))
	lines = append(lines, itemStyle.Render(""))
	lines = append(lines, itemStyle.Render(m.importInput.View()))
	lines = append(lines, itemStyle.Render(""))

	helpText := fmt.Sprintf(" %s Import • %s Cancel",
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderCreateDialog renders the create context dialog
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

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Create Docker Context "))
	lines = append(lines, itemStyle.Render(""))
	lines = append(lines, itemStyle.Render(m.createNameInput.View()))
	lines = append(lines, itemStyle.Render(m.createDescInput.View()))
	lines = append(lines, itemStyle.Render(m.createHostInput.View()))
	lines = append(lines, itemStyle.Render(""))

	// TLS checkbox
	checkbox := "[ ]"
	if m.createTLSEnabled {
		checkbox = "[✓]"
	}
	checkboxStyle := itemStyle
	if m.createInputFocus == 3 {
		checkboxStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("63")).
			Bold(true)
	}
	lines = append(lines, checkboxStyle.Render(checkbox+" Use TLS"))

	// Show cert file inputs only if TLS is enabled
	if m.createTLSEnabled {
		lines = append(lines, itemStyle.Render(""))

		// CA file with browse button indicator
		caLine := m.createCAInput.View()
		if m.createInputFocus == 4 {
			caLine += "  " + keyStyle.Render("[f: Browse]")
		}
		lines = append(lines, itemStyle.Render(caLine))

		// Cert file with browse button indicator
		certLine := m.createCertInput.View()
		if m.createInputFocus == 5 {
			certLine += "  " + keyStyle.Render("[f: Browse]")
		}
		lines = append(lines, itemStyle.Render(certLine))

		// Key file with browse button indicator
		keyLine := m.createKeyInput.View()
		if m.createInputFocus == 6 {
			keyLine += "  " + keyStyle.Render("[f: Browse]")
		}
		lines = append(lines, itemStyle.Render(keyLine))
	}

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, itemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, itemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			keyStyle.Render("<Enter>"),
			keyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Create • %s Navigate • %s Toggle TLS • %s Browse • %s Cancel",
			keyStyle.Render("<Enter>"),
			keyStyle.Render("<Tab/↑/↓>"),
			keyStyle.Render("<Space>"),
			keyStyle.Render("<f>"),
			keyStyle.Render("<Esc>"))
	}
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderEditDialog renders the edit context dialog (description only)
func (m *Model) renderEditDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Edit Context: "+m.editContextName+" "))
	lines = append(lines, itemStyle.Render(""))
	lines = append(lines, itemStyle.Render(m.editDescInput.View()))

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, itemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, itemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			keyStyle.Render("<Enter>"),
			keyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Update • %s Cancel",
			keyStyle.Render("<Enter>"),
			keyStyle.Render("<Esc>"))
	}
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderFileBrowserDialog renders the file browser dialog
func (m *Model) renderFileBrowserDialog() string {
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
	lines = append(lines, titleStyle.Render(fmt.Sprintf(" Select .tar file - Directory: %s ", m.fileBrowserPath)))
	lines = append(lines, itemStyle.Render(""))

	// Show files with cursor
	maxVisible := 10
	start := m.fileBrowserCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.fileBrowserFiles) {
		end = len(m.fileBrowserFiles)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		item := m.fileBrowserFiles[i]
		displayName := ""

		// Handle parent directory
		if item == ".." {
			displayName = "📁 .."
		} else if strings.HasSuffix(item, "/") {
			// Directory
			displayName = "📁 " + filepath.Base(strings.TrimSuffix(item, "/"))
		} else {
			// File
			displayName = filepath.Base(item)
		}

		if i == m.fileBrowserCursor {
			lines = append(lines, selectedStyle.Render("→ "+displayName))
		} else {
			lines = append(lines, itemStyle.Render("  "+displayName))
		}
	}

	lines = append(lines, itemStyle.Render(""))
	helpText := fmt.Sprintf(" %s Select/Navigate • %s / %s Move • %s Cancel",
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<↑/↓>"),
		keyStyle.Render("<PgUp/PgDn>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderCertFileBrowserDialog renders the certificate file browser dialog
func (m *Model) renderCertFileBrowserDialog() string {
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

	fileTypeLabel := ""
	switch m.certFileTarget {
	case "ca":
		fileTypeLabel = "CA Certificate"
	case "cert":
		fileTypeLabel = "Client Certificate"
	case "key":
		fileTypeLabel = "Client Key"
	}

	// Count actual files (excluding "..")
	fileCount := len(m.fileBrowserFiles)
	if fileCount > 0 && m.fileBrowserFiles[0] == ".." {
		fileCount--
	}

	var lines []string
	lines = append(lines, titleStyle.Render(fmt.Sprintf(" Select %s ", fileTypeLabel)))
	lines = append(lines, itemStyle.Render(fmt.Sprintf("Directory: %s (%d files)", m.fileBrowserPath, fileCount)))
	lines = append(lines, itemStyle.Render(""))

	// Show files with cursor
	maxVisible := 10
	start := m.fileBrowserCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.fileBrowserFiles) {
		end = len(m.fileBrowserFiles)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		filePath := m.fileBrowserFiles[i]
		var displayName string
		if filePath == ".." {
			displayName = "📁 .."
		} else {
			displayName = filepath.Base(filePath)
			// Show directory indicator
			if strings.HasSuffix(filePath, "/") {
				displayName = "📁 " + strings.TrimSuffix(displayName, "/")
			}
		}
		if i == m.fileBrowserCursor {
			lines = append(lines, selectedStyle.Render("→ "+displayName))
		} else {
			lines = append(lines, itemStyle.Render("  "+displayName))
		}
	}

	lines = append(lines, itemStyle.Render(""))
	helpText := fmt.Sprintf(" %s Select/Navigate • %s / %s Move • %s Cancel",
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<↑/↓>"),
		keyStyle.Render("<PgUp/PgDn>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderErrorDialog renders the error dialog
func (m *Model) renderErrorDialog() string {
	return errordialog.Render(m.GetError())
}
