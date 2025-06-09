package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
