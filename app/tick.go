package app

import (
	tea "github.com/charmbracelet/bubbletea"
	systeminfoview "swarmcli/views/systeminfo"
	"time"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	return m, systeminfoview.LoadStatus()
}
