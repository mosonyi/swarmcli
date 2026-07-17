// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"strings"

	"swarmcli/docker"
	"swarmcli/views/commandinput"
	"swarmcli/views/confirmdialog"
	loadingview "swarmcli/views/loading"
	"swarmcli/views/searchinput"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/unlockdialog"

	"github.com/charmbracelet/lipgloss"
	"swarmcli/views/view"
	"swarmcli/views/viewstack"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model holds app state
type Model struct {
	deps docker.Deps

	viewport viewport.Model

	systemInfo *systeminfoview.Model

	currentView view.View
	viewStack   viewstack.Stack

	commandInput *commandinput.Model
	searchInput  *searchinput.Model

	// App-level error dialog for loading/navigation errors.
	// Per-view confirmdialog stays for operation errors (scale, restart, etc.).
	errorDialog          *confirmdialog.Model
	appErrorDialogActive bool
	errorFallbackView    string // view to navigate to on dismiss; "" = goBack()

	// App-level swarm unlock-key dialog, shown when the active swarm is locked.
	unlockDialog       *unlockdialog.Model
	unlockDialogActive bool

	// App-level "update available" notice, raised once per new release at
	// startup (and on demand via :dev-update). Driven by the systeminfo
	// version check; dismissal opt-out persists via the settings package.
	updateDialog         *confirmdialog.Model
	updateDialogActive   bool
	pendingUpdateVersion string // version to persist if the opt-out box is ticked

	// Terminal dimensions
	terminalWidth  int
	terminalHeight int

	// Previous context name, set on context switch for revert on failure
	previousContext string

	// Fullscreen mode — hides helpbar/stackbar, uses full terminal
	fullscreen bool
}

var depsTransform func(docker.Deps) docker.Deps

// SetDepsTransform registers a function that transforms the default Deps
// before they are stored in the Model. Must be called before InitialModel().
// Used by extension builds to wrap Deps interfaces with middleware.
func SetDepsTransform(fn func(docker.Deps) docker.Deps) {
	depsTransform = fn
}

func buildDeps() docker.Deps {
	deps := docker.DefaultDeps()
	if depsTransform != nil {
		deps = depsTransform(deps)
	}
	return deps
}

func InitialModel() *Model {
	// Use larger initial dimensions that will be adjusted by first WindowSizeMsg
	// This avoids the loading view appearing too small on the first render
	terminalWidth, terminalHeight := 200, 50

	vp := viewport.New(terminalWidth, terminalHeight)
	// Start viewport below the system info header (original default)
	vp.YPosition = 5

	loading := loadingview.New(terminalWidth, terminalHeight, true, map[string]string{
		"title":   "Initializing",
		"header":  "Fetching cluster info",
		"message": "Loading Swarm nodes and stacks...",
	})

	deps := buildDeps()

	return &Model{
		deps:           deps,
		viewport:       vp,
		currentView:    loading,
		systemInfo:     systeminfoview.New(deps, version, edition),
		viewStack:      viewstack.Stack{},
		commandInput:   cmdBar(),
		searchInput:    searchinput.New(),
		errorDialog:    confirmdialog.New(terminalWidth, terminalHeight),
		unlockDialog:   unlockdialog.New(terminalWidth, terminalHeight),
		updateDialog:   confirmdialog.New(terminalWidth, terminalHeight),
		terminalWidth:  terminalWidth,
		terminalHeight: terminalHeight,
	}
}

// Init  will be automatically called by Bubble Tea if the model implements the Model interface
// and is passed into the tea.NewProgram function.
func (m *Model) Init() tea.Cmd {
	// "" loads all stacks on all nodes
	return tea.Batch(
		tick(),
		loadSnapshotAsync(),
		m.systemInfo.LoadStatus(),
		m.systemInfo.Init(), // Initialize systeminfo's tick commands
		watchEventsCmd(),
	)
}

func (m *Model) switchToView(name string, data any) tea.Cmd {
	factory, ok := view.GetFactory(name)
	if !ok {
		return nil
	}

	// Exit hook for current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.deps, m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height, false)

	// Push current view onto stack
	m.viewStack.Push(m.currentView)
	m.currentView = newView

	// Enter hook for new view
	enterCmd := newView.OnEnter()

	return tea.Batch(exitCmd, resizeCmd, loadCmd, enterCmd)
}

func (m *Model) replaceView(name string, data any) tea.Cmd {
	factory, ok := view.GetFactory(name)
	if !ok {
		return nil
	}

	// Run exit hook on current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.deps, m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height, false)

	m.currentView = newView
	m.viewStack.Reset()

	// Run enter hook on new view
	enterCmd := newView.OnEnter()

	return tea.Batch(exitCmd, resizeCmd, loadCmd, enterCmd)
}

// StackBarSuffix is optional right-aligned text on the breadcrumb bar.
// Set from init() to display persistent status (e.g., license mode).
var StackBarSuffix string

// commandHintText advertises the ":" command bar, which is invisible until
// pressed. ":help" opens the command list; "?" opens the keybindings — the
// wording deliberately keeps those two distinct.
const commandHintText = "Type :help for commands"

// stackBarGap is the minimum number of blank columns kept between the
// breadcrumbs and the right-aligned block, and between the segments inside it.
const stackBarGap = 2

// stackBarMaxCrumbs is the most breadcrumb segments shown before the oldest
// are collapsed into a "…" prefix.
const stackBarMaxCrumbs = 3

var (
	stackBarSuffixStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	stackBarHintStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func (m *Model) renderStackBar() string {
	names := make([]string, 0, m.viewStack.Len()+1)
	for _, v := range m.viewStack.Views() {
		names = append(names, v.Name())
	}
	names = append(names, m.currentView.Name())
	crumbs := m.fitBreadcrumbs(names)

	// Try the right-aligned renderings widest-first and take the first that
	// fits. The hint is dropped before the suffix: the suffix carries
	// persistent status, the hint is only a discoverability aid.
	for _, right := range m.stackBarRight() {
		gap := m.terminalWidth - lipgloss.Width(crumbs) - lipgloss.Width(right)
		if gap >= stackBarGap {
			return crumbs + strings.Repeat(" ", gap) + right
		}
	}
	return crumbs
}

// stackBarRight returns the right-aligned stack bar renderings in preference
// order, widest first. Empty when there is nothing to right-align.
func (m *Model) stackBarRight() []string {
	suffix := ""
	if StackBarSuffix != "" {
		suffix = stackBarSuffixStyle.Render(StackBarSuffix)
	}
	hint := ""
	if !m.hidesGlobalKeys() {
		hint = stackBarHintStyle.Render(commandHintText)
	}

	switch {
	case hint != "" && suffix != "":
		return []string{hint + strings.Repeat(" ", stackBarGap) + suffix, suffix}
	case suffix != "":
		return []string{suffix}
	case hint != "":
		return []string{hint}
	default:
		return nil
	}
}

// fitBreadcrumbs renders the breadcrumb trail for names, shedding the oldest
// segments until the line fits terminalWidth. RenderBreadcrumbs already
// collapses what it drops into a "…" prefix, so narrowing maxDisplay degrades
// gracefully and always keeps the current view — the rightmost segment, and
// the one worth keeping. Truncates only when even a single segment overflows.
func (m *Model) fitBreadcrumbs(names []string) string {
	if m.terminalWidth <= 0 {
		return RenderBreadcrumbs(names, stackBarMaxCrumbs)
	}
	crumbs := ""
	for display := stackBarMaxCrumbs; display >= 1; display-- {
		crumbs = RenderBreadcrumbs(names, display)
		if lipgloss.Width(crumbs) <= m.terminalWidth {
			return crumbs
		}
	}
	return lipgloss.NewStyle().MaxWidth(m.terminalWidth).Render(crumbs)
}

func cmdBar() *commandinput.Model {
	cmdBar := commandinput.New()
	return cmdBar
}

func (m *Model) showAppError(errMsg string, fallbackView string) {
	m.errorDialog.Visible = true
	m.errorDialog.InfoMode = false
	m.errorDialog.ErrorMode = true
	m.errorDialog.Message = errMsg
	m.appErrorDialogActive = true
	m.errorFallbackView = fallbackView
}

// showAppInfo shows the app-level modal styled as a neutral notice (not an error).
func (m *Model) showAppInfo(infoMsg string, fallbackView string) {
	m.errorDialog.Visible = true
	m.errorDialog.ErrorMode = false
	m.errorDialog.InfoMode = true
	m.errorDialog.Message = infoMsg
	m.appErrorDialogActive = true
	m.errorFallbackView = fallbackView
}
