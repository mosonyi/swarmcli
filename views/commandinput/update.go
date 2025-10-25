package commandinput

import tea "github.com/charmbracelet/bubbletea"

// Update handles key events and manages input/history state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {

		case "enter":
			val := m.input.Value()
			if val == "" {
				return m, nil
			}
			m.history = append(m.history, val)
			m.histPos = len(m.history)
			m.input.Reset()
			m.errorMsg = ""
			m.visible = false
			return m, func() tea.Msg { return SubmitMsg{Command: val} }

		case "esc":
			m.Hide()
			return m, nil

		case "up":
			if len(m.history) == 0 {
				break
			}
			if m.histPos > 0 {
				m.histPos--
			}
			m.input.SetValue(m.history[m.histPos])
			m.input.CursorEnd()

		case "down":
			if len(m.history) == 0 {
				break
			}
			if m.histPos < len(m.history)-1 {
				m.histPos++
				m.input.SetValue(m.history[m.histPos])
			} else {
				m.histPos = len(m.history)
				m.input.Reset()
			}
			m.input.CursorEnd()

		default:
			// Clear error when user edits
			if m.errorMsg != "" && (msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace) {
				m.errorMsg = ""
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}
