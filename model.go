package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	"swarmcli/views/stacks"
	systeminfoview "swarmcli/views/systeminfo"
)

type mode string

// Model holds app state
type model struct {
	mode     mode
	view     string // "main" or "nodeStacks"
	viewport viewport.Model
	//commandMode  bool
	//commandInput string

	systemInfo systeminfoview.Model
	nodesV     nodesview.Model
	stacks     stacksview.Model
	logs       logs.Model
	inspect    inspectview.Model
}

// initialModel creates default model
func initialModel() model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	return model{
		mode:       modeNodes,
		viewport:   vp,
		systemInfo: systeminfoview.New(version),
		logs:       logs.New(80, 20),
		inspect:    inspectview.New(80, 20),
		nodesV:     nodesview.New(80, 20),
	}
}
