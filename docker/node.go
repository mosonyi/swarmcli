package docker

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

type StackService struct {
	StackName   string
	ServiceName string
}

func GetNodeIDs() ([]string, error) {
	out, err := exec.Command("docker", "node", "ls", "--format", "{{.ID}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("Error getting nodes: %v\n%s", err, out)
	}
	return strings.Fields(string(out)), nil
}

func GetNodeTaskNames(nodeID string) ([]string, error) {
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

func GetServiceNameToIDMap() (map[string]string, error) {
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

func GetNodeStacks(nodeID string) []StackService {
	taskNames, err := GetNodeTaskNames(nodeID)
	if err != nil {
		return nil
	}

	serviceNames := extractServiceNames(taskNames)
	if len(serviceNames) == 0 {
		return nil
	}

	nameToID, err := GetServiceNameToIDMap()
	if err != nil {
		return nil
	}

	var serviceIDs []string
	for name := range serviceNames {
		if id, ok := nameToID[name]; ok {
			serviceIDs = append(serviceIDs, id)
		}
	}
	if len(serviceIDs) == 0 {
		return nil
	}

	stackServices, err := inspectStackServices(serviceIDs)
	if err != nil {
		return nil
	}

	sort.Slice(stackServices, func(i, j int) bool {
		return stackServices[i].StackName < stackServices[j].StackName
	})

	return stackServices
}

func GetAllStacks() []StackService {
	nodeIDs, err := GetNodeIDs()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	allStacks := make(map[string]StackService)
	for _, nodeID := range nodeIDs {
		nodeStacks := GetNodeStacks(nodeID)
		for _, s := range nodeStacks {
			key := s.StackName + "|" + s.ServiceName
			allStacks[key] = s
		}
	}

	var uniqueStacks []StackService
	for _, s := range allStacks {
		uniqueStacks = append(uniqueStacks, s)
	}

	sort.Slice(uniqueStacks, func(i, j int) bool {
		if uniqueStacks[i].StackName == uniqueStacks[j].StackName {
			return uniqueStacks[i].ServiceName < uniqueStacks[j].ServiceName
		}
		return uniqueStacks[i].StackName < uniqueStacks[j].StackName
	})

	return uniqueStacks
}
