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
		// Get all task names for the node
		out, err := exec.Command("docker", "node", "ps", nodeID, "--format", "{{.Name}}").CombinedOutput()
		if err != nil {
			return Msg{Error: fmt.Sprintf("Error getting node tasks: %v\n%s", err, out)}
		}

		// Extract unique service names
		serviceNames := make(map[string]struct{})
		for _, name := range strings.Fields(string(out)) {
			if parts := strings.SplitN(name, ".", 2); len(parts) > 0 {
				serviceNames[parts[0]] = struct{}{}
			}
		}

		// Map service name to ID
		servicesOut, err := exec.Command("docker", "service", "ls", "--format", "{{.ID}} {{.Name}}").CombinedOutput()
		if err != nil {
			return Msg{Error: fmt.Sprintf("Error getting services: %v\n%s", err, servicesOut)}
		}
		nameToID := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(string(servicesOut)), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				nameToID[parts[1]] = parts[0]
			}
		}

		// Collect relevant service IDs
		var ids []string
		for name := range serviceNames {
			if id, ok := nameToID[name]; ok {
				ids = append(ids, id)
			}
		}
		if len(ids) == 0 {
			return Msg{Services: nil}
		}

		// Inspect relevant services for stack labels
		args := append([]string{"service", "inspect"}, ids...)
		args = append(args, "--format", "{{.ID}} {{ index .Spec.Labels \"com.docker.stack.namespace\" }} {{.Spec.Name}}")
		inspectOut, err := exec.Command("docker", args...).CombinedOutput()
		if err != nil {
			return Msg{Error: fmt.Sprintf("Error inspecting services: %v\n%s", err, inspectOut)}
		}

		// Build stack-service pairs
		unique := make(map[string]struct{})
		var stackServices []StackService
		for _, line := range strings.Split(strings.TrimSpace(string(inspectOut)), "\n") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			key := parts[1] + "|" + parts[2]
			if _, exists := unique[key]; !exists && parts[1] != "" && parts[2] != "" {
				unique[key] = struct{}{}
				stackServices = append(stackServices, StackService{
					StackName:   parts[1],
					ServiceName: parts[2],
				})
			}
		}

		sort.Slice(stackServices, func(i, j int) bool {
			return stackServices[i].StackName < stackServices[j].StackName
		})

		return Msg{Services: stackServices}
	}
}
