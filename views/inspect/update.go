// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"fmt"
	"strings"

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
		m.viewport.Height = msg.Height
		m.ready = true
		m.updateViewport()
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
		// content is directly used, do nothing here
		return
	}
	content := m.renderYAML()
	m.viewport.SetContent(content)
}

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))                                             // blueish keys
var searchHighlightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("11")) // yellow highlight

func (m *Model) renderYAML() string {
	if m.Root == nil {
		return ""
	}

	var build func(n *Node, indent int) []string
	build = func(n *Node, indent int) []string {
		var lines []string
		prefix := strings.Repeat("  ", indent)

		key := n.Key
		value := n.ValueStr

		// highlight search term in key
		if m.SearchTerm != "" {
			key = highlightMatches(key, m.SearchTerm, keyStyle)
		} else {
			key = keyStyle.Render(key)
		}

		// highlight search term in value
		if value != "" && m.SearchTerm != "" {
			value = highlightMatches(value, m.SearchTerm, lipgloss.NewStyle()) // default value style
		}

		line := fmt.Sprintf("%s%s", prefix, key)
		if value != "" {
			line += fmt.Sprintf(": %s", value)
		}
		lines = append(lines, line)

		// recursively render children
		for _, c := range n.Children {
			lines = append(lines, build(c, indent+1)...)
		}
		return lines
	}

	return strings.Join(build(m.Root, 0), "\n")
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
