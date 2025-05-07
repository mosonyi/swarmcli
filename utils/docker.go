package utils

import (
	"os/exec"
	"strings"
)

func ListSwarmNodes() ([]string, error) {
	cmd := exec.Command("docker", "node", "ls", "--format", "{{.ID}}\t{{.Hostname}}\t{{.Status}}\t{{.Availability}}\t{{.ManagerStatus}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

func GetSwarmCPUUsage() string {
	cmd := exec.Command("sh", "-c", `docker stats --no-stream --format '{{.CPUPerc}}' | awk -F '%' '{sum += $1} END {printf "%.1f%%", sum}'`)
	out, err := cmd.Output()
	if err != nil {
		return "0%"
	}
	return strings.TrimSpace(string(out))
}

func GetSwarmMemUsage() string {
	cmd := exec.Command("sh", "-c", `docker stats --no-stream --format '{{.MemPerc}}' | awk -F '%' '{sum += $1} END {printf "%.1f%%", sum}'`)
	out, err := cmd.Output()
	if err != nil {
		return "0%"
	}
	return strings.TrimSpace(string(out))
}

func GetContainerCount() string {
	cmd := exec.Command("sh", "-c", "docker ps -q | wc -l")
	out, err := cmd.Output()
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(string(out))
}

func GetServiceCount() string {
	cmd := exec.Command("sh", "-c", "docker service ls -q | wc -l")
	out, err := cmd.Output()
	if err != nil {
		return "0"
	}
	return strings.TrimSpace(string(out))
}

func GetDockerVersion() string {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
