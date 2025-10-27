package commandinput

import (
	"strings"
	"swarmcli/commands"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyTab:
			prefix := strings.TrimSpace(m.input.Value())
			suggestions := commands.Suggestions(prefix)
			if len(suggestions) == 1 {
				m.input.SetValue(suggestions[0] + " ")
			} else if len(suggestions) > 1 {
				// Optional: show popup or print in status line
				return m, tea.Printf("Suggestions: %v", suggestions)
			}
		}

		switch key := msg.String(); key {
		case "enter":
			raw := strings.TrimSpace(m.input.Value())
			if raw == "" {
				return m, nil
			}

			m.history = append(m.history, raw)
			m.histPos = len(m.history)

			// Parse the command
			parts := strings.Fields(raw)
			name := strings.TrimPrefix(parts[0], m.cmdPrefix)
			args := parts[1:]

			c, ok := m.commands[name]
			if !ok {
				m.errorMsg = "Unknown command: " + name
				return m, nil
			}

			m.input.Reset()
			m.visible = false
			m.errorMsg = ""

			if c.Handler != nil {
				return m, func() tea.Msg { return c.Handler(args) }
			}

			return m, func() tea.Msg { return SubmitMsg{Command: name, Args: args} }

		case "esc":
			m.Hide()
			return m, nil

		case "up":
			if len(m.history) > 0 && m.histPos > 0 {
				m.histPos--
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			}
			return m, nil

		case "down":
			if m.histPos < len(m.history)-1 {
				m.histPos++
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			} else {
				m.histPos = len(m.history)
				m.input.Reset()
			}
			return m, nil

		default:
			if m.errorMsg != "" && (msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace) {
				m.errorMsg = ""
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
