package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"swarmcli/app"
)

func init() {
	app.Init()
}

func main() {
	p := tea.NewProgram(app.InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
