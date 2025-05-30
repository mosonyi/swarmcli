package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
)

type mode string

// Model holds app state
type model struct {
	mode           mode
	view           string // "main" or "nodeStacks"
	items          []string
	cursor         int
	viewport       viewport.Model
	commandMode    bool
	commandInput   string
	selectedNodeID string

	// status overview fields
	host           string
	version        string
	cpuUsage       string
	memUsage       string
	containerCount int
	serviceCount   int

	logs logs.Model

	inspect inspectview.Model
}

// initialModel creates default model
func initialModel() model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	return model{
		mode:     modeNodes,
		viewport: vp,
		logs:     logs.New(80, 20),
		inspect:  inspectview.New(80, 20),
	}
}
