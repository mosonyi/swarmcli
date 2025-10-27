package helpview

import (
	"fmt"
	"strings"
	"swarmcli/views/helpbar"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Visible  bool
	content  string
	commands []CommandInfo
}

type CommandInfo struct {
	Name        string
	Description string
}

func New(width, height int, cmds []CommandInfo) Model {
	var b strings.Builder
	for _, c := range cmds {
		fmt.Fprintf(&b, ":%-15s %s\n", c.Name, c.Description)
	}

	return Model{
		Visible:  true,
		content:  b.String(),
		commands: cmds,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "close"},
	}
}
