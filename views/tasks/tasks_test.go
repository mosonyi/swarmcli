// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	}
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func fakeTasks(names ...string) []docker.TaskEntry {
	tasks := make([]docker.TaskEntry, len(names))
	for i, name := range names {
		tasks[i] = docker.TaskEntry{
			Name:         name,
			ServiceName:  "svc-" + name,
			NodeName:     fmt.Sprintf("node%d", i),
			DesiredState: "running",
			CurrentState: "running",
			Image:        "alpine:latest",
		}
	}
	return tasks
}

func testModel() *Model {
	return New(80, 24, "test-stack")
}

// --- Tests ---

func TestNew(t *testing.T) {
	m := New(80, 24, "mystack")
	require.Equal(t, 80, m.width)
	require.Equal(t, 24, m.height)
	require.Equal(t, "mystack", m.stackName)
	require.Equal(t, SortByName, m.sortField)
	require.True(t, m.sortAscending)
	require.True(t, m.visible)
}

func TestName(t *testing.T) {
	m := testModel()
	require.Equal(t, "tasks", m.Name())
}

func TestHasErrors(t *testing.T) {
	m := testModel()
	require.False(t, m.HasErrors())
}

func TestOnEnter(t *testing.T) {
	m := testModel()
	m.visible = false
	cmd := m.OnEnter()
	require.True(t, m.visible)
	require.NotNil(t, cmd) // LoadTasksCmd
}

func TestOnExit(t *testing.T) {
	m := testModel()
	m.OnExit()
	require.False(t, m.visible)
}

func TestSetSize(t *testing.T) {
	m := testModel()
	m.SetSize(120, 40)
	require.Equal(t, 120, m.width)
	require.Equal(t, 40, m.height)
	require.Equal(t, 120, m.viewport.Width)
	require.Equal(t, 40, m.viewport.Height)
}

func TestShortHelpItems(t *testing.T) {
	m := testModel()
	items := m.ShortHelpItems()
	require.True(t, len(items) >= 5)
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["shift+n"])
	require.True(t, keys["shift+s"])
	require.True(t, keys["shift+d"])
	require.True(t, keys["shift+t"])
}

// --- Rendering tests ---

// The tint rule itself is tested in views/taskutil; what this asserts is that
// the rows here go through it, and that a replica swarm stopped on purpose no
// longer reads like a crash (issue #601). A row with nothing wrong keeps this
// view's own cyan, which is not the colour the other two views use.
func TestRenderTasks_TintsByTaskState(t *testing.T) {
	trueColour(t)
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: []docker.TaskEntry{
		{Name: "web.1", ServiceName: "web", NodeName: "node-up", DesiredState: "running",
			State: "running", CurrentState: "running 18 minutes ago"},
		{Name: "web.2", ServiceName: "web", NodeName: "node-gone", DesiredState: "shutdown",
			State: "shutdown", CurrentState: "shutdown 11 minutes ago"},
		{Name: "web.3", ServiceName: "web", NodeName: "node-bad", DesiredState: "shutdown",
			State: "failed", CurrentState: "failed 19 minutes ago", Error: "task: non-zero exit (1)"},
	}})

	out := m.renderTasks()
	require.Equal(t, fgSeq(ui.ListItemStyle.GetForeground()), rowTint(t, out, "node-up"))
	require.Equal(t, fgSeq(lipgloss.Color("3")), rowTint(t, out, "node-gone"))
	require.Equal(t, fgSeq(lipgloss.Color("9")), rowTint(t, out, "node-bad"))
}

// trueColour makes a test's assertions run against the tinted rows: the default
// profile under `go test` is Ascii, where every style renders as plain text.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// rowTint returns the opening SGR sequence of the rendered row carrying marker.
func rowTint(t *testing.T, out, marker string) string {
	t.Helper()
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(ansi.Strip(line), marker) {
			return sgrPrefix.FindString(line)
		}
	}
	require.FailNowf(t, "no rendered row", "no row containing %q", marker)
	return ""
}

var sgrPrefix = regexp.MustCompile(`^\x1b\[[0-9;]*m`)

// fgSeq is the SGR sequence a foreground colour renders as, so a test names the
// colour it expects rather than an escape sequence.
func fgSeq(c lipgloss.TerminalColor) string {
	return strings.SplitN(lipgloss.NewStyle().Foreground(c).Render("x"), "x", 2)[0]
}

// --- Update tests ---

func TestUpdate_TasksLoaded(t *testing.T) {
	m := testModel()
	cmd := m.Update(TasksLoadedMsg{Tasks: fakeTasks("t1", "t2")})
	require.Len(t, m.tasks, 2)
	// A load does not arm the ticker. Only OnEnter and a tick itself do, so a
	// load issued by OnEnter or the factory cannot start a second chain
	// alongside the one OnEnter already started.
	require.Nil(t, cmd)
}

func TestUpdate_TasksLoaded_Error(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Error: fmt.Errorf("fail")})
	require.Empty(t, m.tasks)
}

func TestUpdate_TickMsg_Visible(t *testing.T) {
	m := testModel()
	m.visible = true
	cmd := m.Update(TickMsg{})
	require.NotNil(t, cmd) // CheckTasksCmd
}

func TestUpdate_TickMsg_NotVisible(t *testing.T) {
	m := testModel()
	m.visible = false
	cmd := m.Update(TickMsg{})
	require.NotNil(t, cmd) // tickCmd (keeps polling)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	require.Equal(t, 100, m.width)
	require.Equal(t, 30, m.height)
}

// --- Sort keys ---

func TestKey_Sort_Name(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("b", "a")})
	require.Equal(t, SortByName, m.sortField)
	require.True(t, m.sortAscending)
	m.Update(key("N"))
	require.False(t, m.sortAscending) // toggled
}

func TestKey_Sort_Service(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("a")})
	m.Update(key("S"))
	require.Equal(t, SortByService, m.sortField)
	require.True(t, m.sortAscending)
}

func TestKey_Sort_Node(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("a")})
	m.Update(key("D"))
	require.Equal(t, SortByNode, m.sortField)
}

func TestKey_Sort_State(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("a")})
	m.Update(key("T"))
	require.Equal(t, SortByState, m.sortField)
}

// The "?" key is routed by the app, not by this view — see app.Model.openHelp
// and its tests. What the view still owns is the content the app asks it for.
func TestHelpContent(t *testing.T) {
	m := testModel()
	require.NotEmpty(t, m.HelpContent())
}

func TestGetTasksHelpContent(t *testing.T) {
	cats := GetTasksHelpContent()
	require.True(t, len(cats) >= 2)
	require.Equal(t, "View", cats[0].Title)
	require.Equal(t, "Navigation", cats[1].Title)
}

// --- Sorting ---

func TestApplySorting_ByName(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("c", "a", "b")})
	m.sortField = SortByName
	m.sortAscending = true
	m.applySorting()
	require.Equal(t, "a", m.tasks[0].Name)
	require.Equal(t, "b", m.tasks[1].Name)
	require.Equal(t, "c", m.tasks[2].Name)
}

func TestApplySorting_ByNameDesc(t *testing.T) {
	m := testModel()
	m.Update(TasksLoadedMsg{Tasks: fakeTasks("a", "c", "b")})
	m.sortField = SortByName
	m.sortAscending = false
	m.applySorting()
	require.Equal(t, "c", m.tasks[0].Name)
}
