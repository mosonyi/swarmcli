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
		// 1. Get all task names for the node
		cmd := exec.Command("docker", "node", "ps", nodeID, "--format", "{{.Name}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return Msg{
				Error: fmt.Sprintf("Error getting node tasks: %v\n%s", err, out),
			}
		}
		taskNames := strings.Fields(string(out))

		// 2. Extract unique service names from task names
		serviceNamesSet := make(map[string]struct{})
		for _, taskName := range taskNames {
			parts := strings.SplitN(taskName, ".", 2)
			if len(parts) > 0 {
				serviceNamesSet[parts[0]] = struct{}{}
			}
		}
		var serviceNames []string
		for name := range serviceNamesSet {
			serviceNames = append(serviceNames, name)
		}

		// 3. Get all services and map name -> ID
		cmd = exec.Command("docker", "service", "ls", "--format", "{{.ID}} {{.Name}}")
		servicesOut, err := cmd.CombinedOutput()
		if err != nil {
			return Msg{Error: fmt.Sprintf("Error getting services: %v\n%s", err, servicesOut)}
		}
		serviceNameToID := make(map[string]string)
		for _, line := range strings.Split(strings.TrimSpace(string(servicesOut)), "\n") {
			parts := strings.Fields(line)
			if len(parts) == 2 {
				serviceNameToID[parts[1]] = parts[0]
			}
		}

		// 4. Collect relevant service IDs
		var relevantIDs []string
		for _, name := range serviceNames {
			if id, ok := serviceNameToID[name]; ok {
				relevantIDs = append(relevantIDs, id)
			}
		}
		if len(relevantIDs) == 0 {
			return Msg{Services: nil}
		}

		// 5. Batch inspect all relevant services for stack labels
		args := append([]string{"service", "inspect"}, relevantIDs...)
		args = append(args, "--format", "{{.ID}} {{ index .Spec.Labels \"com.docker.stack.namespace\" }} {{.Spec.Name}}")
		cmd = exec.Command("docker", args...)
		inspectOut, err := cmd.CombinedOutput()
		if err != nil {
			return Msg{Error: fmt.Sprintf("Error inspecting services: %v\n%s", err, inspectOut)}
		}

		// 6. Build stack-service pairs
		unique := make(map[string]struct{})
		var stackServices []StackService
		for _, line := range strings.Split(strings.TrimSpace(string(inspectOut)), "\n") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			stackName := parts[1]
			serviceName := parts[2]
			if stackName == "" || serviceName == "" {
				continue
			}
			key := stackName + "|" + serviceName
			if _, exists := unique[key]; !exists {
				unique[key] = struct{}{}
				stackServices = append(stackServices, StackService{
					StackName:   stackName,
					ServiceName: serviceName,
				})
			}
		}

		sort.Slice(stackServices, func(i, j int) bool {
			return stackServices[i].StackName < stackServices[j].StackName
		})

		return Msg{Services: stackServices}
	}
}
