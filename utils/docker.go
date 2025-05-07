package utils

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

func ListSwarmNodes() ([]string, error) {
	out, err := RunDockerCmd("node", "ls", "--format", "{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}")
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func GetSwarmCPUUsage() string {
	out, err := RunDockerCmd("stats", "--no-stream", "--format", "{{.CPUPerc}}")
	if err != nil {
		return "0%"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var total float64
	for _, line := range lines {
		// Trim % sign and parse
		value := strings.TrimSuffix(line, "%")
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			total += f
		}
	}

	return fmt.Sprintf("%.1f%%", total)
}

func GetSwarmMemUsage() string {
	out, err := RunDockerCmd("stats", "--no-stream", "--format", "{{.MemPerc}}")
	if err != nil {
		return "0%"
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var total float64
	for _, line := range lines {
		// Remove trailing '%' and whitespace
		value := strings.TrimSuffix(line, "%")
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err == nil {
			total += f
		}
	}

	return fmt.Sprintf("%.1f%%", total)
}

func GetContainerCount() string {
	out, err := RunDockerCmd("ps", "-q")
	if err != nil {
		return "0"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "0"
	}
	return strconv.Itoa(len(lines))
}

func GetServiceCount() string {
	out, err := RunDockerCmd("service", "ls", "-q")
	if err != nil {
		return "0"
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return "0"
	}
	return strconv.Itoa(len(lines))
}

func GetDockerVersion() string {
	out, err := RunDockerCmd("version", "--format", "{{.Server.Version}}")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
