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
		m.viewport.SetContent(m.renderVisible())
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
		// Not valid JSON: show as single-line text node
		rawRoot := &Node{
			Key:      "root",
			Raw:      jsonStr,
			ValueStr: jsonStr,
			Expanded: true,
			Path:     "root",
		}
		m.Root = rawRoot
	} else {
		m.Root = root
		m.Root.Expanded = true
	}
	// rebuild visible and reset cursor
	m.rebuildVisible()
	m.Cursor = 0
	m.searchMode = false
	m.SearchTerm = ""
	m.searchIndex = 0

	if m.ready {
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderVisible())
	}
}

// renderVisible produces the textual content for the viewport and highlights the cursor line
func (m *Model) renderVisible() string {
	if m.Visible == nil || len(m.Visible) == 0 {
		return "▶ root"
	}
	lines := make([]string, 0, len(m.Visible))
	for i, n := range m.Visible {
		prefix := strings.Repeat("  ", n.Depth)
		symbol := "  "
		if len(n.Children) > 0 {
			if n.Expanded {
				symbol = "▼ "
			} else {
				symbol = "▶ "
			}
		}
		lineKey := n.Key
		if n.ValueStr != "" {
			lineKey = fmt.Sprintf("%s: %s", n.Key, n.ValueStr)
		}
		line := fmt.Sprintf("%s%s%s", prefix, symbol, lineKey)
		if i == m.Cursor {
			line = "» " + line
		} else if m.SearchTerm != "" && !n.Matches {
			// dim non-matching lines during search by replacing with a faint prefix (simple approach)
			line = "  " + line
		} else {
			line = "  " + line
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
