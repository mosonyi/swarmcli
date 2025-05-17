package main

import (
	"github.com/charmbracelet/bubbles/viewport"
)

type mode string

// Model holds app state
type model struct {
	mode            mode
	view            string // "main" or "nodeStacks"
	items           []string
	cursor          int
	viewport        viewport.Model
	inspectViewport viewport.Model
	inspecting      bool
	inspectText     string
	commandMode     bool
	commandInput    string
	selectedNodeID  string

	// status overview fields
	host           string
	version        string
	cpuUsage       string
	memUsage       string
	containerCount int
	serviceCount   int

	// node stacks
	nodeStacks  []string
	stackCursor int

	logsViewport        viewport.Model
	viewingLogs         bool
	stackLogsText       string
	stackLogsSearchTerm string
	stackLogsSearchMode bool
	nodeStackLines      []string
	nodeServices        []string

	// Search inside inspect view
	inspectSearchMode bool   // Are we in search mode inside inspect view?
	inspectSearchTerm string // The search term to highlight
}

// initialModel creates default model
func initialModel() model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	inspectVp := viewport.New(80, 20) // initial size, will update on WindowSizeMsg

	return model{
		mode:            modeNodes,
		viewport:        vp,
		inspectViewport: inspectVp,
	}
}
