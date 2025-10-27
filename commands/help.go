package commands

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type HelpCommand struct{}

func (HelpCommand) Name() string        { return "help" }
func (HelpCommand) Description() string { return "Show all available commands" }

func (HelpCommand) Execute(ctx Context, args []string) tea.Cmd {
	cmds := List() // ✅ call directly (no prefix needed)

	var b strings.Builder
	fmt.Fprintf(&b, "\nAvailable commands:\n\n")
	for _, c := range cmds {
		fmt.Fprintf(&b, "  %-20s %s\n", c.Name(), c.Description())
	}
	fmt.Fprintf(&b, "\n")

	return tea.Printf(b.String())
}
