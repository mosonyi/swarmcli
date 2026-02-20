// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"fmt"
	"swarmcli/ui"
	"swarmcli/ui/dialog"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string {
	return fmt.Sprintf("Stacks on Node (Total: %d)", len(m.List.Items))
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

	l().Debugf("Rendering stacks view: createDialogActive=%v step=%s fileBrowserActive=%v",
		m.createDialogActive, m.createDialogStep, m.fileBrowserActive)

	width := frame.FrameWidth
	if m.createDialogActive {
		content = ui.OverlayCentered(content, m.renderCreateDialog(), width, 0)
	} else if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select Compose File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
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

	l().Debugf("renderCreateDialog: step=%s content=%d bytes path=%s",
		m.createDialogStep, len(m.createDialogContent), m.createStackPath)

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

		// Show file path input with browse indicator when focused
		// Always reserve space for the hint to prevent dialog resizing
		fileLine := m.createFileInput.View()
		if m.createInputFocus == 1 {
			fileLine += "  " + dialog.KeyStyle.Render("[f: Browse]")
		} else {
			// Add invisible spacing to maintain consistent width
			fileLine += "             " // 13 characters to match "  [f: Browse]"
		}
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
				dialog.KeyStyle.Render("<f>"),
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
