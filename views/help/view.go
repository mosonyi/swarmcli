// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"strings"

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
	return ui.FrameHeaderStyle.Render("Available Commands")
}

// SupportContact, when non-empty, is rendered as a SUPPORT line at the
// bottom of the keybinding cheat-sheet. Empty by default so OSS shows
// nothing; editions set it to their support address at startup.
var SupportContact string

func (m *Model) FrameFooter() string {
	// overflows is set by the last render, which the app runs before it asks
	// for the footer. Both variants are one line, so the frame's row budget is
	// the same either way and a stale flag cannot resize anything.
	if m.overflows {
		return ui.StatusBarStyle.Render("<↑/↓> scroll · <esc> go back")
	}
	return ui.StatusBarStyle.Render("Press <esc> to go back")
}

func (m *Model) FrameContent() string {
	if m.commandHelp != nil {
		return m.buildCommandHelpContent()
	}
	if len(m.categories) > 0 {
		return m.buildCategorizedContent()
	}
	// Command-list path (`:help`): pin the edition support line at the bottom.
	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(m.Viewable.Width, m.Viewable.Height, m.width, m.height, header, footer)
	return appendSupportLine(m.Viewable.View(), frame.DesiredContentLines)
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(), m.Viewable.Width, m.Viewable.Height, false)
}

// buildCategorizedContent lays the cheat sheet out as category blocks packed
// into as many columns as the terminal has room for, and hands the result to
// the viewport so a screen taller than the frame scrolls instead of losing its
// tail. See layout.go for why the column count is not the category count.
func (m *Model) buildCategorizedContent() string {
	width := m.contentWidth()
	categories := m.filteredCategories()

	// Balancing can find that the sheet reads better in fewer columns than the
	// width allows. Take that answer and lay out again in it, so the columns
	// that remain get the whole width rather than leaving an empty one.
	cols := columnCount(width, len(categories))
	packed := packAt(categories, width, cols)
	for len(packed) < cols && cols > 1 {
		cols = len(packed)
		packed = packAt(categories, width, cols)
	}

	colWidth := width / max(len(packed), 1)
	for i, col := range packed {
		packed[i] = lipgloss.NewStyle().Width(colWidth).Render(col)
	}

	return m.scroll(lipgloss.JoinHorizontal(lipgloss.Top, packed...))
}

// packAt renders the categories for a cols-wide layout and balances them
// across it.
func packAt(categories []HelpCategory, width, cols int) []string {
	colWidth := width / cols
	blocks := make([]string, 0, len(categories))
	for _, cat := range categories {
		blocks = append(blocks, renderCategory(cat, colWidth-categoryGutter))
	}
	return packColumns(blocks, cols)
}

// contentWidth is the width the cheat sheet has to lay out in.
func (m *Model) contentWidth() int {
	if m.Viewable.Width > 0 {
		return m.Viewable.Width
	}
	if m.width > 0 {
		return m.width
	}
	return 80
}

// filteredCategories applies the app-level "/" query, keeping a category only
// while it still has an item that matches.
func (m *Model) filteredCategories() []HelpCategory {
	if m.query == "" {
		return m.categories
	}
	lower := strings.ToLower(m.query)
	var filtered []HelpCategory
	for _, cat := range m.categories {
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
	return filtered
}

// scroll puts content in the viewport, sized to the rows the frame leaves, and
// records whether any of it is out of sight so the footer can say so.
//
// The frame is measured from m.height rather than from the viewport's own
// height, which this then overwrites: reading back what we set would shrink the
// page a little more on every render.
func (m *Model) scroll(content string) string {
	frame := ui.ComputeFrameDimensions(m.contentWidth(), m.height, m.width, m.height, "", m.FrameFooter())
	rows := frame.DesiredContentLines
	if SupportContact != "" {
		// The SUPPORT line and its spacer are pinned below the scrolling part.
		rows = max(0, rows-2)
	}
	if rows < 1 {
		rows = 1
	}

	m.Viewable.Width = m.contentWidth()
	m.Viewable.Height = rows
	m.Viewable.SetContent(content)
	// bubbles keeps YOffset across a height change, so a viewport that grew
	// would pad the rows it gained until the next keypress clamped it.
	m.Viewable.SetYOffset(m.Viewable.YOffset)
	m.overflows = m.Viewable.TotalLineCount() > rows

	out := m.Viewable.View()
	if SupportContact == "" {
		return out
	}
	support := categoryTitleStyle.Render("SUPPORT") + "  " + itemDescStyle.Render(SupportContact)
	return ui.TrimOrPadContentToLines(out+"\n"+support, frame.DesiredContentLines)
}

// appendSupportLine fits content to total lines, pinning the edition SUPPORT
// line (when SupportContact is set) one blank line above the footer so it
// doesn't crowd it. With SupportContact empty (OSS) it just fits content to
// total, rendering nothing extra.
func appendSupportLine(content string, total int) string {
	if SupportContact == "" {
		return ui.TrimOrPadContentToLines(content, total)
	}
	categoryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	// Reserve the SUPPORT line plus one blank spacer above the footer.
	body := ui.TrimOrPadContentToLines(content, max(0, total-2))
	support := categoryStyle.Render("SUPPORT") + "  " + descStyle.Render(SupportContact)
	return ui.TrimOrPadContentToLines(body+"\n"+support, total)
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
