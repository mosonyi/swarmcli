package logs

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport      viewport.Model
	Visible       bool
	searchTerm    string
	searchIndex   int
	searchMatches []int  // indexes of match positions
	mode          string // "normal", "search"
	logLines      string
}

// Create a new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

// Log loading command
func Load(serviceID string) tea.Cmd {
	return func() tea.Msg {
		out, err := exec.Command("docker", "service", "logs", "--no-trunc", serviceID).CombinedOutput()
		if err != nil {
			return Msg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return Msg(out)
	}
}

func (m Model) SetSize(width, height int) Model {
	m.viewport.Width = width
	m.viewport.Height = height - 4 // adjust for borders or header
	return m
}
