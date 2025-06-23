package inspectview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
	"strings"
)

type Model struct {
	viewport      viewport.Model
	Visible       bool
	searchTerm    string
	searchIndex   int
	searchMatches []int  // indexes of match positions
	mode          string // "normal", "search"
	inspectLines  string
	ready         bool
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		mode:     "normal",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Name() string {
	return ViewName
}

func LoadInspectItem(line string) tea.Cmd {
	return func() tea.Msg {
		item := strings.Fields(line)[0]
		var out []byte
		var err error
		out, err = exec.Command("docker", "node", "inspect", item).CombinedOutput()

		if err != nil {
			return Msg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return Msg(out)
	}
}
