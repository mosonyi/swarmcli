package helpview

import (
	"fmt"
	"strings"
	"swarmcli/commands"
	"swarmcli/views/helpbar"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Visible bool
	content string
}

func New(width, height int) Model {
	var b strings.Builder
	for _, cmd := range commands.All() {
		b.WriteString(fmt.Sprintf("  %-20s %s\n", cmd.Name(), cmd.Description()))
	}
	return Model{
		Visible: true,
		content: b.String(),
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
