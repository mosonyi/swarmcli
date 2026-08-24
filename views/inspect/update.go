// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"fmt"
	"strings"

	"github.com/Eldara-Tech/swarmcli/v2/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		m.SetTitle(msg.Title)
		m.SetContent(msg.Content)
		m.ready = true
		return nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		// msg.Height is the frame's height; the viewport only gets what is left
		// after the frame's own rows and the header, so that what it scrolls is
		// exactly what the frame draws.
		m.viewport.Height = max(1, ui.ContentRows(msg.Height, ui.FramedChromeRows, m.FrameHeader(), m.FrameFooter()))
		m.ready = true
		m.updateViewport()
		// A resize changes how many lines fit, but not the offset the viewport
		// scrolls from, so a viewport that grew keeps drawing the lines it drew
		// before and leaves the rows it gained blank.
		if m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		}
		return nil

	case tea.KeyMsg:
		if m.searchMode {
			return handleSearchKey(m, msg)
		}
		return handleNormalKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(content string) {
	m.RawContent = content
	if m.Format == "raw" {
		// raw mode bypasses parsing entirely
		m.Root = nil
		m.viewport.SetContent(content)
		return
	}

	// yml/json mode (existing behaviour)
	root, err := ParseJSON(content)
	if err != nil {
		// fallback
		m.ParseError = err.Error()
		m.SetFormat("raw") // implicit fallback
		return
	}

	m.Root = root
	m.updateViewport()
}

// updateViewport updates viewport content, preserving scroll if possible
func (m *Model) updateViewport() {
	if m.Format == "raw" {
		content := m.RawContent
		if m.filterQuery != "" {
			lower := strings.ToLower(m.filterQuery)
			var filtered []string
			for _, line := range strings.Split(content, "\n") {
				if strings.Contains(strings.ToLower(line), lower) {
					filtered = append(filtered, line)
				}
			}
			content = strings.Join(filtered, "\n")
		}
		if m.SearchTerm != "" {
			content = highlightInContent(content, m.SearchTerm)
		}
		m.viewport.SetContent(content)
		return
	}
	content := m.renderYAML()
	m.viewport.SetContent(content)
}

// renderedLine holds both the plain-text and styled versions of a YAML line.
type renderedLine struct {
	plainText string
	styled    string
}

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))                                             // blueish keys
var searchHighlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("11")) // yellow highlight

func (m *Model) renderYAML() string {
	if m.Root == nil {
		return ""
	}

	// Pass 1: build all lines with plain text and styled versions
	allLines := m.buildYAMLLines(m.Root, 0)

	// Pass 2: filter by filterQuery on plain text
	var visible []renderedLine
	if m.filterQuery != "" {
		lower := strings.ToLower(m.filterQuery)
		for _, rl := range allLines {
			if strings.Contains(strings.ToLower(rl.plainText), lower) {
				visible = append(visible, rl)
			}
		}
	} else {
		visible = allLines
	}

	// Pass 3: join styled lines
	result := make([]string, len(visible))
	for i, rl := range visible {
		result[i] = rl.styled
	}
	return strings.Join(result, "\n")
}

// buildYAMLLines recursively builds renderedLine entries for the YAML tree.
func (m *Model) buildYAMLLines(n *Node, indent int) []renderedLine {
	var lines []renderedLine
	prefix := strings.Repeat("  ", indent)

	key := n.Key
	value := n.ValueStr

	// Build plain text version (no ANSI styles)
	plainLine := prefix + key
	if value != "" {
		plainLine += ": " + value
	}

	// Build styled version
	var styledKey string
	if m.SearchTerm != "" {
		styledKey = highlightMatches(key, m.SearchTerm, keyStyle)
	} else {
		styledKey = keyStyle.Render(key)
	}

	var styledValue string
	if value != "" && m.SearchTerm != "" {
		styledValue = highlightMatches(value, m.SearchTerm, lipgloss.NewStyle())
	} else {
		styledValue = value
	}

	styledLine := fmt.Sprintf("%s%s", prefix, styledKey)
	if value != "" {
		styledLine += fmt.Sprintf(": %s", styledValue)
	}

	lines = append(lines, renderedLine{plainText: plainLine, styled: styledLine})

	for _, c := range n.Children {
		lines = append(lines, m.buildYAMLLines(c, indent+1)...)
	}
	return lines
}

// highlightMatches highlights all occurrences of term in text with yellow background
func highlightMatches(text, term string, style lipgloss.Style) string {
	lowerText := strings.ToLower(text)
	lowerTerm := strings.ToLower(term)

	result := ""
	offset := 0
	for {
		idx := strings.Index(lowerText[offset:], lowerTerm)
		if idx == -1 {
			result += style.Render(text[offset:]) // style remaining
			break
		}
		result += style.Render(text[offset : offset+idx])
		result += searchHighlightStyle.Render(text[offset+idx : offset+idx+len(term)])
		offset += idx + len(term)
	}
	return result
}

// highlightInContent highlights search term in raw content string
func highlightInContent(content, term string) string {
	lowerContent := strings.ToLower(content)
	lowerTerm := strings.ToLower(term)

	result := ""
	offset := 0
	for {
		idx := strings.Index(lowerContent[offset:], lowerTerm)
		if idx == -1 {
			result += content[offset:]
			break
		}
		result += content[offset : offset+idx]
		result += searchHighlightStyle.Render(content[offset+idx : offset+idx+len(term)])
		offset += idx + len(term)
	}
	return result
}

// visiblePlainLines returns the plain-text lines visible after filtering.
func (m *Model) visiblePlainLines() []string {
	if m.Format == "raw" {
		lines := strings.Split(m.RawContent, "\n")
		if m.filterQuery == "" {
			return lines
		}
		lower := strings.ToLower(m.filterQuery)
		var filtered []string
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), lower) {
				filtered = append(filtered, line)
			}
		}
		return filtered
	}

	if m.Root == nil {
		return nil
	}

	allLines := m.buildYAMLLines(m.Root, 0)
	if m.filterQuery != "" {
		lower := strings.ToLower(m.filterQuery)
		var filtered []string
		for _, rl := range allLines {
			if strings.Contains(strings.ToLower(rl.plainText), lower) {
				filtered = append(filtered, rl.plainText)
			}
		}
		return filtered
	}

	result := make([]string, len(allLines))
	for i, rl := range allLines {
		result[i] = rl.plainText
	}
	return result
}

// computeSearchMatches finds all visible lines containing the search term.
func (m *Model) computeSearchMatches() {
	m.searchMatches = nil
	m.searchIndex = 0
	if m.SearchTerm == "" {
		return
	}
	lower := strings.ToLower(m.SearchTerm)
	for i, line := range m.visiblePlainLines() {
		if strings.Contains(strings.ToLower(line), lower) {
			m.searchMatches = append(m.searchMatches, i)
		}
	}
}

// scrollToMatch centers the viewport on the selected match
func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	idx := m.searchMatches[m.searchIndex]
	offset := idx - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.updateViewport()
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
}
