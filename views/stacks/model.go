package stacksview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"os/exec"
	"sort"
	"strings"
)

type Model struct {
	viewport      viewport.Model
	Visible       bool
	stackCursor   int
	stackServices []StackService
	ready         bool
}

type StackService struct {
	StackName   string
	ServiceName string
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

func LoadNodeStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "node", "ps", nodeID, "--format", "{{.Name}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return Msg{
				Error: fmt.Sprintf("Error getting node tasks: %v\n%s", err, out),
			}
		}

		taskNames := strings.Fields(string(out))
		serviceNamesSet := make(map[string]struct{})
		for _, taskName := range taskNames {
			parts := strings.Split(taskName, ".")
			if len(parts) > 0 {
				serviceNamesSet[parts[0]] = struct{}{}
			}
		}

		var stackServices []StackService
		stackSet := make(map[string]struct{})
		for serviceName := range serviceNamesSet {
			cmdServiceID := exec.Command("docker", "service", "ls", "--filter", "name="+serviceName, "--format", "{{.ID}}")
			idOut, err := cmdServiceID.CombinedOutput()
			if err != nil || len(idOut) == 0 {
				continue
			}
			serviceID := strings.TrimSpace(string(idOut))

			cmdInspect := exec.Command("docker", "service", "inspect", serviceID, "--format", "{{ index .Spec.Labels \"com.docker.stack.namespace\" }}")
			stackNameBytes, err := cmdInspect.CombinedOutput()
			if err != nil {
				continue
			}
			stackName := strings.TrimSpace(string(stackNameBytes))
			if stackName != "" {
				stackSet[stackName] = struct{}{}
				stackServices = append(stackServices, StackService{
					StackName:   stackName,
					ServiceName: serviceName,
				})
			}
		}

		// Sort stackServices by StackName for consistent display
		sort.Slice(stackServices, func(i, j int) bool {
			return stackServices[i].StackName < stackServices[j].StackName
		})

		return Msg{Services: stackServices}
	}
}
