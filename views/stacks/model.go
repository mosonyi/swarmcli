package stacks

import (
	"fmt"
	"os/exec"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport       viewport.Model
	Visible        bool
	nodeStacks     []string
	stackCursor    int
	nodeStackLines []string
	nodeServices   []string
	ready          bool
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
