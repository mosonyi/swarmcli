package inspectview

import (
	"fmt"
	"strings"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		// New content arrived
		m.SetTitle(msg.Title)
		m.SetContent(msg.Content)
		m.ready = true
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.ready = true
		// refresh content
		m.viewport.SetContent(m.renderYAML())
		return m, nil

	case tea.KeyMsg:
		// handle keys
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

func (m *Model) updateViewport() {
	content := m.renderYAML()
	m.viewport.SetContent(content)
	if m.ready {
		m.viewport.GotoTop()
	}
}

// renderYAML formats the tree as indented YAML-like text with keys highlighted
func (m *Model) renderYAML() string {
	if m.Root == nil {
		return ""
	}

	var build func(n *Node, indent int) []string
	build = func(n *Node, indent int) []string {
		var lines []string
		prefix := strings.Repeat("  ", indent)
		key := keyStyle.Render(n.Key)

		if n.ValueStr != "" || len(n.Children) == 0 {
			line := fmt.Sprintf("%s%s: %s", prefix, key, n.ValueStr)
			if m.SearchTerm != "" && !strings.Contains(strings.ToLower(line), strings.ToLower(m.SearchTerm)) {
				line = prefix + line // dim non-matching? could add subtle style
			}
			lines = append(lines, line)
		} else {
			lines = append(lines, fmt.Sprintf("%s%s:", prefix, key))
			for _, c := range n.Children {
				lines = append(lines, build(c, indent+1)...)
			}
		}
		return lines
	}

	return strings.Join(build(m.Root, 0), "\n")
}
