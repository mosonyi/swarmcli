package view

import tea "github.com/charmbracelet/bubbletea"

type Factory func(width, height int, payload any) (View, tea.Cmd)
