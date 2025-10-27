package docker

import (
	"fmt"
	"os/exec"
	"strings"
)

// RunDocker runs a docker CLI command and returns trimmed output lines.
func RunDocker(args ...string) ([]string, error) {
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

// RunDockerSingle runs a docker command and returns the raw output string.
func RunDockerSingle(args ...string) (string, error) {
	lines, err := RunDocker(args...)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

// ---------- Common Structs ----------

type SwarmNode struct {
	ID            string
	Hostname      string
	Status        string
	Availability  string
	ManagerStatus string
}

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

type Stack struct {
	Name         string
	Services     string
	Orchestrator string
}

func (s Stack) String() string {
	return strings.Join(StructFieldsAsStringArray(s), " ")
}

// ---------- Query Functions ----------

func ListSwarmNodes() ([]SwarmNode, error) {
	lines, err := RunDocker("node", "ls", "--format", "{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}")
	if err != nil {
		return nil, err
	}

	nodes := make([]SwarmNode, 0, len(lines))
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 5 {
			continue
		}
		nodes = append(nodes, SwarmNode{
			ID:            parts[0],
			Hostname:      parts[1],
			Status:        parts[2],
			Availability:  parts[3],
			ManagerStatus: parts[4],
		})
	}
	return nodes, nil
}

func GetSwarmCPUUsage() string {
	lines, err := RunDocker("stats", "--no-stream", "--format", "{{.CPUPerc}}")
	if err != nil {
		return "0%"
	}
	total := parseAndSumPercentLines(lines)
	return fmt.Sprintf("%.1f%%", total)
}

func GetSwarmMemUsage() string {
	lines, err := RunDocker("stats", "--no-stream", "--format", "{{.MemPerc}}")
	if err != nil {
		return "0%"
	}
	total := parseAndSumPercentLines(lines)
	return fmt.Sprintf("%.1f%%", total)
}

func GetContainerCount() int {
	lines, err := RunDocker("ps", "-q")
	if err != nil {
		return 0
	}
	return countNonEmptyLines(lines)
}

func GetServiceCount() int {
	lines, err := RunDocker("service", "ls", "-q")
	if err != nil {
		return 0
	}
	return countNonEmptyLines(lines)
}

func GetDockerVersion() string {
	version, err := RunDockerSingle("version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(version)
}
