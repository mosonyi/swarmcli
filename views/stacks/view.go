// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/ui/dialog"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string {
	return ui.ScopedTitleFiltered("Stacks", "all", len(m.List.Filtered), m.List.Query)
}

func (m *Model) FrameHeader() string { return m.List.RenderHeader() }
func (m *Model) FrameFooter() string { return m.List.RenderFooter() }

func (m *Model) FrameContent() string {
	m.setRenderItem()

	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(
		m.List.Viewport.Width, m.List.Viewport.Height,
		m.List.Viewport.Width, m.List.Viewport.Height,
		header, footer,
	)
	content := m.List.VisibleContent(frame.DesiredContentLines)

	width := frame.FrameWidth
	if m.createDialogActive {
		content = ui.OverlayCentered(content, m.renderCreateDialog(), width, 0)
	} else if m.saveDialogActive {
		content = ui.OverlayCentered(content, m.renderSaveDialog(), width, 0)
	} else if m.fileBrowserActive {
		title := "Select Compose File"
		if m.fileBrowserContext == "save" {
			title = "Select Save Directory"
		}
		fileBrowserDialog := ui.RenderFileBrowserDialog(title, m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		content = ui.OverlayCentered(content, fileBrowserDialog, width, 0)
	} else if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, 0)
	}

	return content
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.List.Viewport.Width, m.List.Viewport.Height, false)
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialog.TitleStyle.Render(" Create Stack - Choose Source "))
		lines = append(lines, dialog.ItemStyle.Render(""))
		lines = append(lines, dialog.ItemStyle.Render("How would you like to create the stack?"))
		lines = append(lines, dialog.ItemStyle.Render(""))

		if m.createStackSource == "file" {
			lines = append(lines, dialog.SelectedStyle.Render("→ From compose file"))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  From compose file"))
		}

		if m.createStackSource == "inline" {
			lines = append(lines, dialog.SelectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, dialog.ItemStyle.Render("  Inline editor"))
		}

		lines = append(lines, dialog.ItemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<↑>"),
			dialog.KeyStyle.Render("<↓>"),
			dialog.KeyStyle.Render("<Esc>"))
		lines = append(lines, dialog.HelpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, dialog.TitleStyle.Render(" Create Stack from Compose File "))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1).
				Width(70)
			// Wrap error message to multiple lines if needed
			wrappedError := ui.WrapText("⚠ "+m.createDialogError, 68) // 70 - 2 for padding
			for _, line := range wrappedError {
				lines = append(lines, errorStyle.Render(line))
			}
			lines = append(lines, dialog.ItemStyle.Render(""))
		}

		lines = append(lines, dialog.ItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show file path input with browse indicator. The key is a chord, so it
		// works from any focus and the hint is always shown.
		fileLine := m.createFileInput.View() + "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
		lines = append(lines, dialog.ItemStyle.Render(fileLine))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Deploy • %s Navigate • %s Browse • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render(dialog.BrowseHelpKey),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialog.TitleStyle.Render(" Create Stack - Inline Editor "))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1).
				Width(70)
			// Wrap error message to multiple lines if needed
			wrappedError := ui.WrapText("⚠ "+m.createDialogError, 68) // 70 - 2 for padding
			for _, line := range wrappedError {
				lines = append(lines, errorStyle.Render(line))
			}
			lines = append(lines, dialog.ItemStyle.Render(""))
		}

		lines = append(lines, dialog.ItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Show source file if content was loaded from a file
		if m.createStackPath != "" {
			sourceStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)
			lines = append(lines, sourceStyle.Render(fmt.Sprintf("Source: %s", m.createStackPath)))
			lines = append(lines, dialog.ItemStyle.Render(""))
		}

		// Show editor status with edit hint when focused
		// Always reserve space for the hint to prevent dialog resizing
		editorStatus := "Compose YAML: "
		if m.createDialogContent != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createDialogContent))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + dialog.KeyStyle.Render("[e: Edit]")
		} else {
			// Add invisible spacing to maintain consistent width
			editorStatus += "           " // 11 characters to match "  [e: Edit]"
		}
		lines = append(lines, dialog.ItemStyle.Render(editorStatus))
		lines = append(lines, dialog.ItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Deploy • %s Navigate • %s Edit • %s Cancel",
				dialog.KeyStyle.Render("<Enter>"),
				dialog.KeyStyle.Render("<Tab>"),
				dialog.KeyStyle.Render("<e>"),
				dialog.KeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialog.HelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
}

func (m *Model) renderSaveDialog() string {
	var lines []string
	lines = append(lines, dialog.TitleStyle.Render(fmt.Sprintf(" Save Stack: %s ", m.saveStackName)))
	lines = append(lines, dialog.ItemStyle.Render(""))

	if m.saveDialogError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Padding(0, 1).
			Width(70)
		wrappedError := ui.WrapText("⚠ "+m.saveDialogError, 68)
		for _, line := range wrappedError {
			lines = append(lines, errorStyle.Render(line))
		}
		lines = append(lines, dialog.ItemStyle.Render(""))
	}

	fileLine := m.saveFileInput.View()
	fileLine += "  " + dialog.KeyStyle.Render(dialog.BrowseHint)
	lines = append(lines, dialog.ItemStyle.Render(fileLine))
	lines = append(lines, dialog.ItemStyle.Render(""))

	var helpText string
	if m.saveDialogError != "" {
		helpText = fmt.Sprintf(" %s Fix error • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render("<Esc>"))
	} else {
		helpText = fmt.Sprintf(" %s Save • %s Browse • %s Cancel",
			dialog.KeyStyle.Render("<Enter>"),
			dialog.KeyStyle.Render(dialog.BrowseHelpKey),
			dialog.KeyStyle.Render("<Esc>"))
	}
	lines = append(lines, dialog.HelpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialog.BorderStyle.Render(content)
}
