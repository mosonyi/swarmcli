// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"fmt"
	"testing"

	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
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

// --- Update tests ---

func TestUpdate_TasksLoaded(t *testing.T) {
	m := testModel()
	cmd := m.Update(TasksLoadedMsg{Tasks: fakeTasks("t1", "t2")})
	require.Len(t, m.tasks, 2)
	require.NotNil(t, cmd) // tickCmd
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

func TestKey_Help(t *testing.T) {
	m := testModel()
	cmd := m.Update(key("?"))
	require.NotNil(t, cmd)
	msg := cmd()
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, view.NameHelp, nav.ViewName)
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
