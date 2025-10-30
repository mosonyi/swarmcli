package inspectview

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/views/view"
)

var matchStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // blueish

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.SetTitle(msg.Title)
		m.SetContent(msg.Content)
		m.ready = true
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.ready = true
		m.updateViewport()
		return m, nil

	case tea.KeyMsg:
		if m.searchMode {
			return handleSearchKey(m, msg)
		}
		return handleNormalKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetContent(jsonStr string) {
	root, err := ParseJSON(jsonStr)
	if err != nil {
		root = &Node{
			Key:      "root",
			ValueStr: jsonStr,
		}
	}
	m.Root = root
	m.updateViewport()
}

// updateViewport updates viewport content, preserving scroll if possible
func (m *Model) updateViewport() {
	content := m.renderYAML()
	m.viewport.SetContent(content)
}

// renderYAML formats the tree as YAML-like text and highlights matches
func (m *Model) renderYAML() string {
	if m.Root == nil {
		return ""
	}

	var build func(n *Node, indent int) []string
	build = func(n *Node, indent int) []string {
		var lines []string
		prefix := strings.Repeat("  ", indent)
		key := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Render(n.Key)
		line := ""
		if len(n.Children) == 0 || n.ValueStr != "" {
			line = fmt.Sprintf("%s%s: %s", prefix, key, n.ValueStr)
		} else {
			line = fmt.Sprintf("%s%s:", prefix, key)
		}

		// highlight if matches search term
		if m.SearchTerm != "" && strings.Contains(strings.ToLower(line), strings.ToLower(m.SearchTerm)) {
			line = matchStyle.Render(line)
		}

		lines = append(lines, line)

		for _, c := range n.Children {
			lines = append(lines, build(c, indent+1)...)
		}

		return lines
	}

	return strings.Join(build(m.Root, 0), "\n")
}
