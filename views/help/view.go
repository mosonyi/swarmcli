// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string { return "Help" }

func (m *Model) FrameHeader() string {
	if len(m.categories) > 0 {
		return ""
	}
	return ui.FrameHeaderStyle.Render("Available Commands")
}

func (m *Model) FrameFooter() string {
	return ui.StatusBarStyle.Render("Press <esc> to go back")
}

func (m *Model) FrameContent() string {
	if len(m.categories) > 0 {
		return m.buildCategorizedContent()
	}
	// Command help content
	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(m.Viewable.Width, m.Viewable.Height, m.width, m.height, header, footer)
	return ui.TrimOrPadContentToLines(m.Viewable.View(), frame.DesiredContentLines)
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(), m.Viewable.Width, m.Viewable.Height, false)
}

func (m *Model) buildCategorizedContent() string {
	width := m.Viewable.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}

	categoryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2"))

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	// Apply query filter to categories
	categories := m.categories
	if m.query != "" {
		lower := strings.ToLower(m.query)
		var filtered []HelpCategory
		for _, cat := range categories {
			var items []HelpItem
			for _, item := range cat.Items {
				if strings.Contains(strings.ToLower(item.Keys), lower) ||
					strings.Contains(strings.ToLower(item.Description), lower) {
					items = append(items, item)
				}
			}
			if len(items) > 0 {
				filtered = append(filtered, HelpCategory{Title: cat.Title, Items: items})
			}
		}
		categories = filtered
	}

	numCols := len(categories)
	if numCols == 0 {
		numCols = 1
	}
	colWidth := width / numCols
	maxKeyWidth := 15

	maxRows := 0
	for _, cat := range categories {
		if len(cat.Items) > maxRows {
			maxRows = len(cat.Items)
		}
	}

	var headerParts []string
	for _, cat := range categories {
		titleText := strings.ToUpper(cat.Title)
		paddedTitle := fmt.Sprintf("%-*s", colWidth, titleText)
		styledTitle := categoryStyle.Render(paddedTitle)
		headerParts = append(headerParts, styledTitle)
	}
	headerRow := strings.Join(headerParts, "")

	var contentLines []string
	for row := 0; row < maxRows; row++ {
		var rowParts []string
		for _, cat := range categories {
			if row < len(cat.Items) {
				item := cat.Items[row]
				styledKey := keyStyle.Render(fmt.Sprintf("%-*s", maxKeyWidth, item.Keys))
				styledDesc := descStyle.Render(item.Description)

				plainText := fmt.Sprintf("%-*s %s", maxKeyWidth, item.Keys, item.Description)
				plainTextWidth := lipgloss.Width(plainText)

				if plainTextWidth > colWidth {
					descWidth := colWidth - maxKeyWidth - 1
					if descWidth > 0 && len(item.Description) > descWidth {
						styledDesc = descStyle.Render(item.Description[:descWidth-3] + "...")
						plainText = fmt.Sprintf("%-*s %s", maxKeyWidth, item.Keys, item.Description[:descWidth-3]+"...")
						plainTextWidth = lipgloss.Width(plainText)
					}
				}

				styledLine := styledKey + " " + styledDesc
				paddingNeeded := colWidth - plainTextWidth
				if paddingNeeded > 0 {
					styledLine += strings.Repeat(" ", paddingNeeded)
				}

				rowParts = append(rowParts, styledLine)
			} else {
				rowParts = append(rowParts, strings.Repeat(" ", colWidth))
			}
		}
		contentLines = append(contentLines, strings.Join(rowParts, ""))
	}

	fullContent := headerRow + "\n\n" + strings.Join(contentLines, "\n")

	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(m.Viewable.Width, m.Viewable.Height, m.width, m.height, "", footer)
	return ui.TrimOrPadContentToLines(fullContent, frame.DesiredContentLines)
}
