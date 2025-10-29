package inspectview

import (
	"strings"
	"swarmcli/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.mode == "search" {
		return handleSearchModeKey(m, msg)
	}
	return handleNormalModeKey(m, msg)
}

func handleNormalModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.Visible = false
	case "/":
		m.mode = "search"
		m.searchTerm = ""
		m.searchMatches = nil
		m.searchIndex = 0
	case "n":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
		}
	case "N":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
		}
	case "up":
		m.viewport.ScrollUp(1)
	case "down":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	case " ":
		// toggle expand/collapse for current line
		if m.inspectRoot == nil {
			break
		}
		line := m.viewport.YOffset
		path := lineToPath(m.inspectRoot, line, 0, m.expanded)
		if path != "" {
			m.expanded[path] = !m.expanded[path]
			m.inspectLines = strings.Join(RenderTree(m.inspectRoot, 0, m.expanded), "\n")
			m.viewport.SetContent(m.buildContent())
		}
	}
	return m, nil
}

func handleSearchModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		m.searchTerm += msg.String()
		m.searchMatches = utils.FindAllMatches(m.inspectLines, m.searchTerm)
	case tea.KeyBackspace:
		if len(m.searchTerm) > 0 {
			m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
			m.searchMatches = utils.FindAllMatches(m.inspectLines, m.searchTerm)
		}
	case tea.KeyEnter:
		m.mode = "normal"
		m.searchIndex = 0
	case tea.KeyEsc:
		m.mode = "normal"
	}
	m.viewport.SetContent(m.buildContent())
	return m, nil
}
