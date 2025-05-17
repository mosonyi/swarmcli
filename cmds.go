package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"sort"
	"strings"
	"swarmcli/docker"
	"time"
)

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadData(m mode) tea.Cmd {
	return func() tea.Msg {
		var list []string
		switch m {
		case modeNodes:
			nodes, _ := docker.ListSwarmNodes()
			for _, n := range nodes {
				list = append(list, fmt.Sprint(n))
			}
		case modeServices:
			services, _ := docker.ListSwarmServices()
			for _, s := range services {
				list = append(list, fmt.Sprint(s))
			}
		case modeStacks:
			stacks, _ := docker.ListStacks()
			for _, s := range stacks {
				list = append(list, fmt.Sprint(s))
			}
		}
		return loadedMsg(list)
	}
}

func loadStatus() tea.Cmd {
	return func() tea.Msg {
		// Use your docker package functions here
		cpu := docker.GetSwarmCPUUsage()
		mem := docker.GetSwarmMemUsage()
		containers := docker.GetContainerCount()
		services := docker.GetServiceCount()

		host, _ := os.Hostname()
		return statusMsg{
			host:       host,
			version:    version,
			cpu:        cpu,
			mem:        mem,
			containers: containers,
			services:   services,
		}
	}
}

func inspectItem(mode mode, line string) tea.Cmd {
	return func() tea.Msg {
		item := strings.Fields(line)[0]
		var out []byte
		var err error
		switch mode {
		case modeNodes:
			out, err = exec.Command("docker", "node", "inspect", item).CombinedOutput()
		case modeServices:
			out, err = exec.Command("docker", "service", "inspect", item).CombinedOutput()
		case modeStacks:
			out, err = exec.Command("docker", "stack", "services", item).CombinedOutput()
		}
		if err != nil {
			return inspectMsg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return inspectMsg(string(out))
	}
}

func loadNodeStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "node", "ps", nodeID, "--format", "{{.Name}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nodeStacksMsg{
				output: fmt.Sprintf("Error getting node tasks: %v\n%s", err, out),
				stacks: nil,
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
			}
		}

		var stacks []string
		for stack := range stackSet {
			stacks = append(stacks, stack)
		}

		var services []string
		for service := range serviceNamesSet {
			services = append(services, service)
		}

		sort.Strings(stacks)
		var sb strings.Builder
		sb.WriteString("Stacks running on node " + nodeID + ":\n")
		for _, s := range stacks {
			sb.WriteString("- " + s + "\n")
		}

		return nodeStacksMsg{output: sb.String(), stacks: stacks, services: services}
	}
}
