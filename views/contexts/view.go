// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/ui/components/errordialog"
	"github.com/Eldara-Tech/swarmcli/ui/dialog"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string { return "Docker Contexts" }

func (m *Model) FrameHeader() string {
	header := ""
	if m.IsLoading() {
		header = "Loading contexts..."
	} else if m.IsSwitchPending() {
		header = "Switching context..."
	} else if err := m.GetError(); err != "" {
		header = fmt.Sprintf("Error: %s", err)
	} else if msg := m.GetSuccess(); msg != "" {
		header = msg
	} else {
		return m.List.RenderHeader()
	}
	return ui.FrameHeaderStyle.Render(header)
}

func (m *Model) FrameFooter() string { return "" }

func (m *Model) FrameContent() string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	headerRendered := m.FrameHeader()
	footerRendered := m.FrameFooter()

	frame := ui.ComputeFrameDimensions(
		m.viewport.Width, m.viewport.Height,
		m.viewport.Width, m.viewport.Height,
		headerRendered, footerRendered,
	)

	m.List.Viewport.Width = width
	m.List.Viewport.Height = frame.DesiredContentLines

	colWidths := m.List.ColWidths()

	m.List.RenderItem = func(ctx docker.ContextInfo, selected bool, _ int) string {
		current := " "
		if ctx.Current {
			current = "*"
		}
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

		line := fmt.Sprintf("%-*s%-*s%-*s%-*s%-*s",
			colWidths[0], firstCol,
			colWidths[1], tlsChar,
			colWidths[2], desc,
			colWidths[3], host,
			colWidths[4], errStr,
		)
		if selected {
			return ui.ListSelectedStyle.Render(line)
		}
		return ui.ListItemStyle.Render(line)
	}

	var content string
	if m.IsLoading() {
		loadingLine := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("Loading contexts...")
		content = ui.TrimOrPadContentToLines(loadingLine, frame.DesiredContentLines)
	} else {
		content = m.List.VisibleContent(frame.DesiredContentLines)
	}

	if m.certFileBrowserActive {
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

	return content
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// Contexts view has a dynamic header that changes based on list loading state
	headerRendered := m.FrameHeader()
	footerRendered := m.FrameFooter()

	frame := ui.ComputeFrameDimensions(
		m.viewport.Width, m.viewport.Height,
		m.viewport.Width, m.viewport.Height,
		headerRendered, footerRendered,
	)

	m.List.Viewport.Width = width
	m.List.Viewport.Height = frame.DesiredContentLines

	// Get content via FrameContent (which also sets up RenderItem)
	content := m.FrameContent()

	// After FrameContent, header may have changed to list header
	if !m.IsLoading() {
		headerRendered = m.List.RenderHeader()
	}

	return ui.RenderFramedBox(
		m.FrameTitle(),
		headerRendered,
		content,
		footerRendered,
		frame.FrameWidth,
	)
}

func (m *Model) renderImportDialog() string {
	contentWidth := 60

	titleStyleWithWidth := dialog.TitleStyle.Width(contentWidth)
	itemStyleWithWidth := dialog.ItemStyle.Width(contentWidth)
	borderStyleWithWidth := dialog.BorderStyle.Width(contentWidth + dialog.BoxInsetColumns)
	helpStyleWithWidth := dialog.HelpStyle.Width(contentWidth)

	var lines []string
	lines = append(lines, titleStyleWithWidth.Render(" Import Docker Context "))
	lines = append(lines, itemStyleWithWidth.Render("Enter the path to the context tar file:"))
	lines = append(lines, itemStyleWithWidth.Render(""))
	lines = append(lines, itemStyleWithWidth.Render(m.importInput.View()))
	lines = append(lines, itemStyleWithWidth.Render(""))

	helpText := fmt.Sprintf(" %s Confirm • %s Cancel",
		dialog.KeyStyle.Render("<Enter>"),
		dialog.KeyStyle.Render("<Esc>"))
	lines = append(lines, helpStyleWithWidth.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyleWithWidth.Render(content)
}

// renderCreateDialog renders the create context dialog
func (m *Model) renderCreateDialog() string {
	var lines []string
	lines = append(lines, dialog.TitleStyle.Render(" Create Docker Context "))
	lines = append(lines, dialog.ItemStyle.Render(""))
	lines = append(lines, dialog.ItemStyle.Render(m.createNameInput.View()))
	lines = append(lines, dialog.ItemStyle.Render(m.createDescInput.View()))
	lines = append(lines, dialog.ItemStyle.Render(m.createHostInput.View()))
	lines = append(lines, dialog.ItemStyle.Render(""))

	// TLS checkbox
	checkbox := "[ ]"
	if m.createTLSEnabled {
		checkbox = "[✓]"
	}
	checkboxStyle := dialog.ItemStyle
	if m.createInputFocus == 3 {
		checkboxStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("63")).
			Bold(true)
	}
	lines = append(lines, checkboxStyle.Render(checkbox+" Use TLS"))

	// Show cert file inputs only if TLS is enabled
	if m.createTLSEnabled {
		lines = append(lines, dialog.ItemStyle.Render(""))

		// CA file with browse button indicator
		caLine := m.createCAInput.View()
		if m.createInputFocus == 4 {
			caLine += "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
		}
		lines = append(lines, dialog.ItemStyle.Render(caLine))

		// Cert file with browse button indicator
		certLine := m.createCertInput.View()
		if m.createInputFocus == 5 {
			certLine += "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
		}
		lines = append(lines, dialog.ItemStyle.Render(certLine))

		// Key file with browse button indicator
		keyLine := m.createKeyInput.View()
		if m.createInputFocus == 6 {
			keyLine += "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
		}
		lines = append(lines, dialog.ItemStyle.Render(keyLine))
	}

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, dialog.ItemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, dialog.ItemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<Esc>"))
	} else {
		// No Browse entry here: this line already sets the dialog's width, and
		// the per-field hint advertises the key exactly when it is usable.
		helpText = fmt.Sprintf(" %s Create • %s Navigate • %s Toggle TLS • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<Tab/↑/↓>"),
			dialog.KeyStyle.Render("<Space>"),
			dialog.KeyStyle.Render("<Esc>"))
	}
	lines = append(lines, dialog.HelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
}

// renderEditDialog renders the edit context dialog (description and host)
func (m *Model) renderEditDialog() string {
	var lines []string
	lines = append(lines, dialog.TitleStyle.Render(" Edit Context: "+m.editContextName+" "))
	lines = append(lines, dialog.ItemStyle.Render(""))
	lines = append(lines, dialog.ItemStyle.Render(m.editDescInput.View()))
	lines = append(lines, dialog.ItemStyle.Render(m.editHostInput.View()))

	// Show error message if present
	errorMsg := m.GetError()
	if errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1)
		lines = append(lines, dialog.ItemStyle.Render(""))
		lines = append(lines, errorStyle.Render(errorMsg))
	}

	lines = append(lines, dialog.ItemStyle.Render(""))

	// Adjust help text based on whether error is shown
	var helpText string
	if errorMsg != "" {
		helpText = fmt.Sprintf(" %s Clear Error • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Update • %s Navigate • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<Tab/↑/↓>"),
			dialog.KeyStyle.Render("<Esc>"))
	}
	lines = append(lines, dialog.HelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
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
	lines = append(lines, dialog.TitleStyle.Render(fmt.Sprintf(" Select %s ", fileTypeLabel)))
	lines = append(lines, dialog.ItemStyle.Render(fmt.Sprintf("Directory: %s (%d files)", m.fileBrowserPath, fileCount)))
	lines = append(lines, dialog.ItemStyle.Render(""))

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
			lines = append(lines, dialog.SelectedStyle.Render("→ "+displayName))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  "+displayName))
		}
	}

	lines = append(lines, dialog.ItemStyle.Render(""))
	helpText := fmt.Sprintf(" %s Select/Navigate • %s / %s Move • %s Cancel",
		dialog.KeyStyle.Render("<Enter>"),
		dialog.KeyStyle.Render("<↑/↓>"),
		dialog.KeyStyle.Render("<PgUp/PgDn>"),
		dialog.KeyStyle.Render("<Esc>"))
	lines = append(lines, dialog.HelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
}

// renderErrorDialog renders the error dialog
func (m *Model) renderErrorDialog() string {
	return errordialog.Render(m.GetError())
}
