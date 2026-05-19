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
	if m.commandHelp != nil {
		return ui.FrameHeaderStyle.Render(m.commandHelp.Title)
	}
	if len(m.categories) > 0 {
		return ""
	}
	tip := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render("Tip: <command> --help (or -h) for flags, usage & examples")
	return ui.FrameHeaderStyle.Render("Available Commands") + "\n" + tip
}

func (m *Model) FrameFooter() string {
	return ui.StatusBarStyle.Render("Press <esc> to go back")
}

func (m *Model) FrameContent() string {
	if m.commandHelp != nil {
		return m.buildCommandHelpContent()
	}
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

// buildCommandHelpContent renders the per-command help as vertically
// stacked sections in a single column. Short-key sections (flags) use an
// aligned key/description column; sections whose widest key exceeds half
// the width (usage, examples) fall back to stacked key-then-description
// lines so long strings wrap cleanly instead of colliding.
func (m *Model) buildCommandHelpContent() string {
	width := m.Viewable.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}

	categoryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	const indent = "  "
	var b strings.Builder

	for i, cat := range m.commandHelp.Sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(categoryStyle.Render(strings.ToUpper(cat.Title)))
		b.WriteString("\n")

		keyW := 0
		for _, it := range cat.Items {
			if w := lipgloss.Width(it.Keys); w > keyW {
				keyW = w
			}
		}
		// The USAGE block (first section) always stacks key-then-desc so
		// the command line and its description read as a header.
		stacked := i == 0 || keyW > width/2

		for _, it := range cat.Items {
			if stacked || it.Description == "" {
				for _, ln := range wrapText(it.Keys, width-len(indent)) {
					b.WriteString(indent + keyStyle.Render(ln) + "\n")
				}
				if it.Description != "" {
					for _, ln := range wrapText(it.Description, width-len(indent)-2) {
						b.WriteString(indent + "  " + descStyle.Render(ln) + "\n")
					}
				}
				continue
			}

			descW := width - len(indent) - keyW - 2
			if descW < 10 {
				descW = 10
			}
			dl := wrapText(it.Description, descW)
			b.WriteString(indent + keyStyle.Render(fmt.Sprintf("%-*s", keyW, it.Keys)) + "  " + descStyle.Render(dl[0]) + "\n")
			for _, extra := range dl[1:] {
				b.WriteString(indent + strings.Repeat(" ", keyW) + "  " + descStyle.Render(extra) + "\n")
			}
		}

		// Detail prose belongs to the USAGE block (first section):
		// a blank line then the wrapped paragraph(s), author newlines
		// preserved as paragraph breaks.
		if i == 0 && m.commandHelp.Detail != "" {
			b.WriteString("\n")
			for _, raw := range strings.Split(m.commandHelp.Detail, "\n") {
				if strings.TrimSpace(raw) == "" {
					b.WriteString("\n")
					continue
				}
				for _, ln := range wrapText(raw, width-len(indent)) {
					b.WriteString(indent + descStyle.Render(ln) + "\n")
				}
			}
		}
	}

	fullContent := strings.TrimRight(b.String(), "\n")
	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(m.Viewable.Width, m.Viewable.Height, m.width, m.height, header, footer)
	return ui.TrimOrPadContentToLines(fullContent, frame.DesiredContentLines)
}

// wrapText word-wraps s to at most width columns, never splitting a word.
func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	if width <= 0 {
		return []string{strings.Join(words, " ")}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if lipgloss.Width(cur)+1+lipgloss.Width(w) > width {
			lines = append(lines, cur)
			cur = w
		} else {
			cur += " " + w
		}
	}
	return append(lines, cur)
}
