package docker

import (
	"strings"
)

// ---------- Node/Service Utilities ----------

func GetNodeIDs() ([]string, error) {
	lines, err := RunDocker("node", "ls", "--format", "{{.ID}}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(strings.Join(lines, " ")), nil
}

func GetNodeTaskNames(nodeID string) ([]string, error) {
	lines, err := RunDocker("node", "ps", nodeID, "--format", "{{.Name}}")
	if err != nil {
		return nil, err
	}
	return strings.Fields(strings.Join(lines, " ")), nil
}

func extractServiceNames(taskNames []string) map[string]struct{} {
	serviceNames := make(map[string]struct{}, len(taskNames))
	for _, name := range taskNames {
		if parts := strings.SplitN(name, ".", 2); len(parts) > 0 && parts[0] != "" {
			serviceNames[parts[0]] = struct{}{}
		}
	}
	return serviceNames
}

func GetServiceNameToIDMap() (map[string]string, error) {
	lines, err := RunDocker("service", "ls", "--format", "{{.ID}} {{.Name}}")
	if err != nil {
		return nil, err
	}

	nameToID := make(map[string]string, len(lines))
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			nameToID[parts[1]] = parts[0]
		}
	}
	return nameToID, nil
}

func inspectStackServices(serviceIDs []string) ([]StackService, error) {
	if len(serviceIDs) == 0 {
		return nil, nil
	}

	args := append([]string{"service", "inspect"}, serviceIDs...)
	args = append(args, "--format", "{{.ID}} {{ index .Spec.Labels \"com.docker.stack.namespace\" }} {{.Spec.Name}}")

	lines, err := RunDocker(args...)
	if err != nil {
		return nil, err
	}

	unique := make(map[string]struct{})
	stackServices := make([]StackService, 0, len(lines))

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		key := parts[1] + "|" + parts[2]
		if _, exists := unique[key]; !exists {
			unique[key] = struct{}{}
			stackServices = append(stackServices, StackService{
				StackName:   parts[1],
				ServiceName: parts[2],
			})
		}
	}
	return stackServices, nil
}
