package view

import tea "github.com/charmbracelet/bubbletea"

type NavigateToMsg struct {
	ViewName string
	Payload  any // Can be service ID, stack ID, etc.
}

type NavigateBackMsg struct{}

type Navigator interface {
	NavigateTo(name string, payload any) tea.Cmd
}
