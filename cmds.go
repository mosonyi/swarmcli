package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"swarmcli/docker"
	"time"
)

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
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
