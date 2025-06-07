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
		taskNames, err := getNodeTaskNames(nodeID)
		if err != nil {
			return Msg{Error: err.Error()}
		}

		serviceNames := extractServiceNames(taskNames)
		if len(serviceNames) == 0 {
			return Msg{Services: nil}
		}

		nameToID, err := getServiceNameToIDMap()
		if err != nil {
			return Msg{Error: err.Error()}
		}

		var serviceIDs []string
		for name := range serviceNames {
			if id, ok := nameToID[name]; ok {
				serviceIDs = append(serviceIDs, id)
			}
		}
		if len(serviceIDs) == 0 {
			return Msg{Services: nil}
		}

		stackServices, err := inspectStackServices(serviceIDs)
		if err != nil {
			return Msg{Error: err.Error()}
		}

		sort.Slice(stackServices, func(i, j int) bool {
			return stackServices[i].StackName < stackServices[j].StackName
		})

		return Msg{Services: stackServices}
	}
}

func getNodeTaskNames(nodeID string) ([]string, error) {
	out, err := exec.Command("docker", "node", "ps", nodeID, "--format", "{{.Name}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting node tasks: %v\n%s", err, out)
	}
	return strings.Fields(string(out)), nil
}

func extractServiceNames(taskNames []string) map[string]struct{} {
	serviceNames := make(map[string]struct{})
	for _, name := range taskNames {
		if parts := strings.SplitN(name, ".", 2); len(parts) > 0 {
			serviceNames[parts[0]] = struct{}{}
		}
	}
	return serviceNames
}

func getServiceNameToIDMap() (map[string]string, error) {
	out, err := exec.Command("docker", "service", "ls", "--format", "{{.ID}} {{.Name}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting services: %v\n%s", err, out)
	}
	nameToID := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			nameToID[parts[1]] = parts[0]
		}
	}
	return nameToID, nil
}

func inspectStackServices(serviceIDs []string) ([]StackService, error) {
	args := append([]string{"service", "inspect"}, serviceIDs...)
	args = append(args, "--format", "{{.ID}} {{ index .Spec.Labels \"com.docker.stack.namespace\" }} {{.Spec.Name}}")
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error inspecting services: %v\n%s", err, out)
	}
	unique := make(map[string]struct{})
	var stackServices []StackService
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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
	return stackServices, nil
}
