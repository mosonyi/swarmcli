package logsview

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
	ready         bool
}

// Create a new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
		mode:     "normal",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []string {
	if m.mode == "search" {
		return []string{
			"enter: confirm",
			"esc: cancel",
			"n/N: next/prev",
		}
	}
	return []string{
		"/: search",
		"n/N: next/prev",
		"q: close",
	}
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
