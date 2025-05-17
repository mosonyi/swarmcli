package logs

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"swarmcli/utils"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.logLines = string(msg)
		m.viewport.SetContent(m.logLines)
		m.Visible = true

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 2
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {

		case "q", "esc":
			m.Visible = false
			return m, nil

		case "/":
			m.mode = "search"
			m.searchTerm = ""
			return m, nil

		case "enter":
			if m.mode == "search" {
				m.mode = "normal"
				m.searchMatches = utils.FindAllMatches(m.viewport.View(), m.searchTerm)
				m.searchIndex = 0
				m.scrollToMatch()
			}
			return m, nil

		case "n":
			if len(m.searchMatches) > 0 {
				m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
				m.scrollToMatch()
			}
			return m, nil

		case "N":
			if len(m.searchMatches) > 0 {
				m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
				m.scrollToMatch()
			}
			return m, nil

		case "up":
			m.viewport.LineUp(1)
			return m, nil

		case "down":
			m.viewport.LineDown(1)
			return m, nil

		case "pgup":
			m.viewport.ScrollUp(m.viewport.Height)
			return m, nil

		case "pgdown":
			m.viewport.ScrollDown(m.viewport.Height)
			return m, nil

		default:
			if m.mode == "search" {
				if msg.Type == tea.KeyRunes {
					m.searchTerm += msg.String()
				} else if msg.Type == tea.KeyBackspace && len(m.searchTerm) > 0 {
					m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
				}
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetContent(content string) {
	m.viewport.SetContent(content)
	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"
}

func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	matchPos := m.searchMatches[m.searchIndex]
	lines := strings.Split(m.viewport.View()[:matchPos], "\n")
	offset := len(lines) - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
}
