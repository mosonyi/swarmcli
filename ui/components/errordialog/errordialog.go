// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package errordialog

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"github.com/Eldara-Tech/swarmcli/v2/ui/dialog"

	"github.com/charmbracelet/lipgloss"
)

// Render renders an error dialog with the given error message
func Render(errorMsg string) string {
	titleStyle := dialog.TitleStyle.Background(dialog.AccentError)
	borderStyle := dialog.BorderStyle.BorderForeground(dialog.AccentError)

	var lines []string
	lines = append(lines, titleStyle.Render(" Error "))
	lines = append(lines, dialog.ItemStyle.Render(""))

	maxWidth := 70
	wrappedLines := ui.WrapText(errorMsg, maxWidth)
	for _, line := range wrappedLines {
		lines = append(lines, dialog.ItemStyle.Render(line))
	}

	lines = append(lines, dialog.ItemStyle.Render(""))
	helpText := fmt.Sprintf("%s %s %s",
		dialog.HelpStyle.Render("Press"),
		dialog.KeyStyle.Render("<Enter>"),
		dialog.HelpStyle.Render("to close"))
	lines = append(lines, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}
