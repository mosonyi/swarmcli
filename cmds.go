package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"strings"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
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
			return inspectview.Msg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return inspectview.Msg(out)
	}
}
