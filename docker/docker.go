package docker

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func RunDockerCmd(name string, arg ...string) ([]byte, error) {
	args := append([]string{name}, arg...)
	return exec.Command("docker", args...).Output()
}

type SwarmNode struct {
	ID            string
	Hostname      string
	Status        string
	Availability  string
	ManagerStatus string
}

// Remove the braces when printing as this is what we need.
// It might make sense to just use a special method instead
// naively overwriting the fmt.
func (s SwarmNode) String() string {
	return strings.Join(StructFieldsAsStringArray(s), " ")
}

type SwarmService struct {
	Name     string
	Mode     string
	Replicas string
}

func (s SwarmService) String() string {
	return strings.Join(StructFieldsAsStringArray(s), " ")
}

type DockerStack struct {
	Name         string
	Services     string
	Orchestrator string
}

func (s DockerStack) String() string {
	return strings.Join(StructFieldsAsStringArray(s), " ")
}

func ListSwarmNodes() ([]SwarmNode, error) {
	out, err := RunDockerCmd("node", "ls", "--format", "{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}")
	if err != nil {
		return nil, err
	}

	var nodes []SwarmNode
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 5 {
			nodes = append(nodes, SwarmNode{
				ID:            parts[0],
				Hostname:      parts[1],
				Status:        parts[2],
				Availability:  parts[3],
				ManagerStatus: parts[4],
			})
		}
	}
	return nodes, nil
}

func ListSwarmServices() ([]SwarmService, error) {
	out, err := RunDockerCmd("service", "ls", "--format", "{{.Name}}\t{{.Mode}}\t{{.Replicas}}")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var services []SwarmService

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			services = append(services, SwarmService{
				Name:     parts[0],
				Mode:     parts[1],
				Replicas: parts[2],
			})
		}
	}

	return services, nil
}

func ListStacks() ([]DockerStack, error) {
	out, err := RunDockerCmd("stack", "ls", "--format", "{{.Name}}\t{{.Services}}\t{{.Orchestrator}}")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var stacks []DockerStack

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 3 {
			stacks = append(stacks, DockerStack{
				Name:         parts[0],
				Services:     parts[1],
				Orchestrator: parts[2],
			})
		}
	}

	return stacks, nil
}

func GetSwarmCPUUsage() string {
	out, err := RunDockerCmd("stats", "--no-stream", "--format", "{{.CPUPerc}}")
	if err != nil {
		return "0%"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	total := parseAndSumPercentLines(lines)
	return fmt.Sprintf("%.1f%%", total)
}

func GetSwarmMemUsage() string {
	out, err := RunDockerCmd("stats", "--no-stream", "--format", "{{.MemPerc}}")
	if err != nil {
		return "0%"
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	total := parseAndSumPercentLines(lines)
	return fmt.Sprintf("%.1f%%", total)
}

func GetContainerCount() string {
	out, err := RunDockerCmd("ps", "-q")
	if err != nil {
		return "0"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strconv.Itoa(countNonEmptyLines(lines))
}

func GetServiceCount() string {
	out, err := RunDockerCmd("service", "ls", "-q")
	if err != nil {
		return "0"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	return strconv.Itoa(countNonEmptyLines(lines))
}

func GetDockerVersion() string {
	out, err := RunDockerCmd("version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
