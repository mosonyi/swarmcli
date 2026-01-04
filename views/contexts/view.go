package contexts

import (
	"fmt"
	"path/filepath"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"
	"swarmcli/ui/components/errordialog"

	"github.com/charmbracelet/lipgloss"
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

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	title := "Docker Contexts"
	// Default header is empty; only show it for loading/switch/error/success
	header := ""

	if m.IsLoading() {
		header = "Loading contexts..."
	} else if m.IsSwitchPending() {
		header = "Switching context..."
	} else if err := m.GetError(); err != "" {
		header = fmt.Sprintf("Error: %s", err)
	} else if msg := m.GetSuccess(); msg != "" {
		header = msg
	}

	// We'll render the column header in the frame header slot so it
	// appears directly under the top border and aligns with content.
	headerRendered := ui.FrameHeaderStyle.Render(header)

	// Use FilterableList for the contexts content. Keep its viewport
	// size in sync with the containing viewport and compute padding
	// so the framed box fills the area.
	m.List.Viewport.Width = width
	// Use the viewport height directly - it's already adjusted by the app
	// for systeminfo header and breadcrumb
	frameHeight := m.viewport.Height
	if frameHeight <= 0 {
		frameHeight = 20
	}
	m.List.Viewport.Height = frameHeight

	// Compute column widths as five equal percentage chunks so columns
	// start at 0%, 20%, 40%, 60% and 80% of the content width.
	contentWidth := width
	base := contentWidth / 5
	colWidths := make([]int, 5)
	for i := 0; i < 5; i++ {
		colWidths[i] = base
	}
	// Distribute remainder to the leftmost columns
	rem := contentWidth - base*5
	for i := 0; i < rem && i < 5; i++ {
		colWidths[i]++
	}

	// Now define render using those exact column widths so items align
	m.List.RenderItem = func(ctx docker.ContextInfo, selected bool, _ int) string {
		current := " "
		if ctx.Current {
			current = "*"
		}

		// Prepare name with room for the current marker and a space
		nameMax := colWidths[0] - 2
		if nameMax < 0 {
			nameMax = 0
		}
		name := ctx.Name
		if len(name) > nameMax {
			if nameMax > 3 {
				name = name[:nameMax-3] + "..."
			} else {
				name = name[:nameMax]
			}
		}
		firstCol := fmt.Sprintf("%s %s", current, name)

		tlsChar := " "
		if ctx.TLS {
			tlsChar = "●"
		}

		descMax := colWidths[2]
		if descMax < 0 {
			descMax = 0
		}
		desc := ctx.Description
		if len(desc) > descMax {
			if descMax > 3 {
				desc = desc[:descMax-3] + "..."
			} else {
				desc = desc[:descMax]
			}
		}

		hostMax := colWidths[3]
		if hostMax < 0 {
			hostMax = 0
		}
		host := ctx.DockerHost
		if len(host) > hostMax {
			if hostMax > 3 {
				host = host[:hostMax-3] + "..."
			} else {
				host = host[:hostMax]
			}
		}

		errMax := colWidths[4]
		if errMax < 0 {
			errMax = 0
		}
		errStr := ctx.Error
		if len(errStr) > errMax {
			if errMax > 3 {
				errStr = errStr[:errMax-3] + "..."
			} else {
				errStr = errStr[:errMax]
			}
		}

		// Build the line with exact column widths and no extra spacing so
		// each column starts at the expected percent positions.
		line := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
			colWidths[0], firstCol,
			colWidths[1], tlsChar,
			colWidths[2], desc,
			colWidths[3], host,
			colWidths[4], errStr,
		)
		if selected {
			return lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("230")).Render(line)
		}
		return line
	}

	// If we're still loading, show a loading placeholder to avoid flashing
	// "No items found." before the async load completes.
	var content string
	if m.IsLoading() {
		// Show an explicit loading line inside the framed box so users
		// see progress immediately instead of a blank area.
		loadingLine := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Loading contexts...")
		m.List.Viewport.SetContent(loadingLine)
		content = m.List.Viewport.View()
	} else {
		// Column header displayed in the frame header slot. Build it here
		// so it aligns exactly with the row layout.
		listContent := m.List.View()

		// Header: reserve two leading spaces so the NAME label lines up
		// with the name text that starts after the current marker and a space.
		headerLine := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
			colWidths[0], "  NAME",
			colWidths[1], "TLS",
			colWidths[2], "DESCRIPTION",
			colWidths[3], "ENDPOINT",
			colWidths[4], "ERROR",
		)
		headerRendered = ui.FrameHeaderStyle.Render(headerLine)

		content = listContent
	}
	frameWidth := width + 4

	// Overlay dialogs on content BEFORE framing
	if m.certFileBrowserActive {
		// Cert file browser has highest priority when open
		certFileBrowserDialog := m.renderCertFileBrowserDialog()
		content = ui.OverlayCentered(content, certFileBrowserDialog, width, 0)
	} else if m.createDialogActive {
		createDialog := m.renderCreateDialog()
		content = ui.OverlayCentered(content, createDialog, width, 0)
	} else if m.editDialogActive {
		editDialog := m.renderEditDialog()
		content = ui.OverlayCentered(content, editDialog, width, 0)
	} else if m.errorDialogActive {
		errorDialog := m.renderErrorDialog()
		content = ui.OverlayCentered(content, errorDialog, width, 0)
	} else if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select .tar file", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		content = ui.OverlayCentered(content, fileBrowserDialog, width, 0)
	} else if m.importInputActive {
		importDialog := m.renderImportDialog()
		content = ui.OverlayCentered(content, importDialog, width, 0)
	} else if m.confirmDialog.Visible {
		dialogView := ui.RenderConfirmDialog(m.confirmDialog.Message)
		content = ui.OverlayCentered(content, dialogView, width, 0)
	}

	rendered := ui.RenderFramedBox(
		title,
		headerRendered,
		content,
		"",
		frameWidth,
	)

	return rendered
}

func (m *Model) renderImportDialog() string {
	contentWidth := 60

	titleStyleWithWidth := dialogTitleStyle.Width(contentWidth)
	itemStyleWithWidth := dialogItemStyle.Width(contentWidth)
	borderStyleWithWidth := dialogBorderStyle.Width(contentWidth + 2)
	helpStyleWithWidth := dialogHelpStyle.Width(contentWidth)

	var lines []string
	lines = append(lines, titleStyleWithWidth.Render(" Import Docker Context "))
	lines = append(lines, itemStyleWithWidth.Render("Enter the path to the context tar file:"))
	lines = append(lines, itemStyleWithWidth.Render(""))
	lines = append(lines, itemStyleWithWidth.Render(m.importInput.View()))
	lines = append(lines, itemStyleWithWidth.Render(""))

	helpText := fmt.Sprintf(" %s Confirm • %s Cancel",
		dialogKeyStyle.Render("<Enter>"),
		dialogKeyStyle.Render("<Esc>"))
	lines = append(lines, helpStyleWithWidth.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyleWithWidth.Render(content)
}

// renderCreateDialog renders the create context dialog
func (m *Model) renderCreateDialog() string {
	var lines []string
	lines = append(lines, dialogTitleStyle.Render(" Create Docker Context "))
	lines = append(lines, dialogItemStyle.Render(""))
	lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
	lines = append(lines, dialogItemStyle.Render(m.createDescInput.View()))
	lines = append(lines, dialogItemStyle.Render(m.createHostInput.View()))
	lines = append(lines, dialogItemStyle.Render(""))

	// TLS checkbox
	checkbox := "[ ]"
	if m.createTLSEnabled {
		checkbox = "[✓]"
	}
	checkboxStyle := dialogItemStyle
	if m.createInputFocus == 3 {
		checkboxStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("63")).
			Bold(true)
	}
	lines = append(lines, checkboxStyle.Render(checkbox+" Use TLS"))

	// Show cert file inputs only if TLS is enabled
	if m.createTLSEnabled {
		lines = append(lines, dialogItemStyle.Render(""))

		// CA file with browse button indicator
		caLine := m.createCAInput.View()
		if m.createInputFocus == 4 {
			caLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		}
		lines = append(lines, dialogItemStyle.Render(caLine))

		// Cert file with browse button indicator
		certLine := m.createCertInput.View()
		if m.createInputFocus == 5 {
			certLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		}
		lines = append(lines, dialogItemStyle.Render(certLine))

		// Key file with browse button indicator
		keyLine := m.createKeyInput.View()
		if m.createInputFocus == 6 {
			keyLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		}
		lines = append(lines, dialogItemStyle.Render(keyLine))
	}

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, dialogItemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, dialogItemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Create • %s Navigate • %s Toggle TLS • %s Browse • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<Tab/↑/↓>"),
			dialogKeyStyle.Render("<Space>"),
			dialogKeyStyle.Render("<f>"),
			dialogKeyStyle.Render("<Esc>"))
	}
	lines = append(lines, dialogHelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
}

// renderEditDialog renders the edit context dialog (description only)
func (m *Model) renderEditDialog() string {
	var lines []string
	lines = append(lines, dialogTitleStyle.Render(" Edit Context: "+m.editContextName+" "))
	lines = append(lines, dialogItemStyle.Render(""))
	lines = append(lines, dialogItemStyle.Render(m.editDescInput.View()))

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, dialogItemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, dialogItemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Update • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<Esc>"))
	}
	lines = append(lines, dialogHelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
}

// renderCertFileBrowserDialog renders the certificate file browser dialog
func (m *Model) renderCertFileBrowserDialog() string {
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
	lines = append(lines, dialogTitleStyle.Render(fmt.Sprintf(" Select %s ", fileTypeLabel)))
	lines = append(lines, dialogItemStyle.Render(fmt.Sprintf("Directory: %s (%d files)", m.fileBrowserPath, fileCount)))
	lines = append(lines, dialogItemStyle.Render(""))

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
			lines = append(lines, dialogSelectedStyle.Render("→ "+displayName))
		} else {
			lines = append(lines, dialogItemStyle.Render("  "+displayName))
		}
	}

	lines = append(lines, dialogItemStyle.Render(""))
	helpText := fmt.Sprintf(" %s Select/Navigate • %s / %s Move • %s Cancel",
		dialogKeyStyle.Render("<Enter>"),
		dialogKeyStyle.Render("<↑/↓>"),
		dialogKeyStyle.Render("<PgUp/PgDn>"),
		dialogKeyStyle.Render("<Esc>"))
	lines = append(lines, dialogHelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
}

// renderErrorDialog renders the error dialog
func (m *Model) renderErrorDialog() string {
	return errordialog.Render(m.GetError())
}
