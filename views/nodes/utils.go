package nodesview

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
	"strings"
	inspectview "swarmcli/views/inspect"
)

func inspectItem(line string) tea.Cmd {
	return func() tea.Msg {
		item := strings.Fields(line)[0]
		var out []byte
		var err error
		out, err = exec.Command("docker", "node", "inspect", item).CombinedOutput()

		if err != nil {
			return inspectview.Msg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return inspectview.Msg(out)
	}
}
