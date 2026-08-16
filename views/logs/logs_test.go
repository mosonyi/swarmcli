// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Eldara-Tech/swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/docker/api/types/swarm"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func testModel() *Model {
	svc := docker.ServiceEntry{
		ServiceID:   "svc-123",
		ServiceName: "web",
	}
	m := New(80, 24, 1000, svc)
	m.ready = true
	// Provide noop snapshot ops for extractUniqueNodes
	m.deps = docker.Deps{
		Snapshot: &mockSnapshotOps{},
	}
	return m
}

type mockSnapshotOps struct{ snap *docker.SwarmSnapshot }

func (m *mockSnapshotOps) GetSnapshot() *docker.SwarmSnapshot              { return m.snap }
func (m *mockSnapshotOps) SetSnapshot(_ *docker.SwarmSnapshot)             {}
func (m *mockSnapshotOps) InvalidateSnapshot()                             {}
func (m *mockSnapshotOps) RefreshSnapshot() (*docker.SwarmSnapshot, error) { return nil, nil }
func (m *mockSnapshotOps) RefreshSnapshotAsync()                           {}
func (m *mockSnapshotOps) TriggerRefreshIfNeeded()                         {}
func (m *mockSnapshotOps) GetOrRefreshSnapshot() (*docker.SwarmSnapshot, error) {
	return nil, nil
}

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "ctrl+f":
		return tea.KeyMsg{Type: tea.KeyCtrlF}
	}
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// --- Model tests ---

func TestNew(t *testing.T) {
	svc := docker.ServiceEntry{ServiceID: "id", ServiceName: "svc"}
	m := New(80, 24, 500, svc)
	require.Equal(t, 500, m.MaxLines)
	require.False(t, m.Visible)
	require.Equal(t, "normal", m.mode)
	require.True(t, m.getFollow())
	require.True(t, m.getWrap())
}

func TestName(t *testing.T) {
	m := testModel()
	require.Equal(t, "logs", m.Name())
}

func TestHasErrors(t *testing.T) {
	m := testModel()
	require.False(t, m.HasErrors())
}

func TestIsSearching_Normal(t *testing.T) {
	m := testModel()
	require.False(t, m.IsSearching())
}

func TestIsSearching_SearchMode(t *testing.T) {
	m := testModel()
	m.mode = "search"
	require.True(t, m.IsSearching())
}

func TestGetNodeSelectVisible(t *testing.T) {
	m := testModel()
	require.False(t, m.GetNodeSelectVisible())
	m.setNodeSelectVisible(true)
	require.True(t, m.GetNodeSelectVisible())
}

func TestFollowToggle(t *testing.T) {
	m := testModel()
	require.True(t, m.getFollow())
	m.setFollow(false)
	require.False(t, m.getFollow())
}

func TestWrapToggle(t *testing.T) {
	m := testModel()
	require.True(t, m.getWrap())
	m.setWrap(false)
	require.False(t, m.getWrap())
}

func TestNodeFilter(t *testing.T) {
	m := testModel()
	require.Equal(t, "", m.getNodeFilter())
	m.setNodeFilter("node1")
	require.Equal(t, "node1", m.getNodeFilter())
}

func TestShortHelpItems_NormalMode(t *testing.T) {
	m := testModel()
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["ctrl+f"])
	require.True(t, keys["s"])
	require.True(t, keys["w"])
	require.True(t, keys["o"])
	require.True(t, keys["t"])
	require.True(t, keys["esc"])
}

func TestShortHelpItems_SearchMode(t *testing.T) {
	m := testModel()
	m.mode = "search"
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["enter"])
	require.True(t, keys["esc"])
}

func TestShortHelpItems_NoWrapShowsArrows(t *testing.T) {
	m := testModel()
	m.setWrap(false)
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["←/→"])
}

// --- Update tests ---

func TestUpdate_InitStreamMsg(t *testing.T) {
	m := testModel()
	lines := make(chan string, 1)
	errs := make(chan error, 1)
	cmd := m.Update(InitStreamMsg{Lines: lines, Errs: errs, MaxLines: 100})
	require.True(t, m.Visible)
	require.NotNil(t, cmd) // readOneLineCmd
	close(lines)
	close(errs)
}

func TestUpdate_LineMsg(t *testing.T) {
	m := testModel()
	m.Visible = true
	lines := make(chan string, 10)
	errs := make(chan error, 1)
	m.linesChan = lines
	m.errChan = errs

	// Add a line
	m.Update(LineMsg{Line: "node1\x00task-1\x00hello world"})

	m.mu.Lock()
	require.Len(t, m.lines, 1)
	require.Equal(t, "hello world", m.lines[0])
	require.Equal(t, "node1", m.lineNodes[0])
	require.Equal(t, "task-1", m.lineTasks[0])
	m.mu.Unlock()
	close(lines)
	close(errs)
}

func TestUpdate_LineMsg_Trimming(t *testing.T) {
	m := testModel()
	m.MaxLines = 3
	m.Visible = true
	lines := make(chan string, 10)
	errs := make(chan error, 1)
	m.linesChan = lines
	m.errChan = errs

	for i := 0; i < 5; i++ {
		m.Update(LineMsg{Line: "line"})
	}
	m.mu.Lock()
	require.Len(t, m.lines, 3)
	require.Len(t, m.lineTasks, 3) // parallel slice trimmed in lock-step
	require.Len(t, m.lineNodes, 3)
	m.mu.Unlock()
	close(lines)
	close(errs)
}

func TestUpdate_StreamErrMsg(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(StreamErrMsg{Err: nil})
	m.mu.Lock()
	require.Contains(t, m.lines[len(m.lines)-1], "Error")
	m.mu.Unlock()
}

func TestUpdate_StreamDoneMsg(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(StreamDoneMsg{})
	m.mu.Lock()
	require.Contains(t, m.lines[len(m.lines)-1], "stream closed")
	m.mu.Unlock()
}

func TestUpdate_WrapToggledMsg(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.lines = []string{"test line"}
	m.mu.Unlock()
	m.Update(WrapToggledMsg{})
	// Should not panic and viewport should be updated
}

func TestUpdate_NodeFilterToggledMsg(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.lines = []string{"test"}
	m.mu.Unlock()
	m.Update(NodeFilterToggledMsg{})
	// Should not panic
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.viewport.Width)
	// The message carries the frame's height; the viewport gets the rows left
	// after the two border rows and the one-line header.
	require.Equal(t, 37, m.viewport.Height)
	require.Len(t, strings.Split(m.FrameContent(), "\n"), 37)
	require.True(t, m.ready)
}

// --- Key handling: normal mode ---

func TestKey_CtrlF_EntersSearch(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("ctrl+f"))
	require.Equal(t, "search", m.mode)
	require.Equal(t, "", m.searchTerm)
}

func TestKey_S_TogglesFollow(t *testing.T) {
	m := testModel()
	m.Visible = true
	require.True(t, m.getFollow())
	m.Update(key("s"))
	require.False(t, m.getFollow())
	m.Update(key("s"))
	require.True(t, m.getFollow())
}

func TestKey_W_TogglesWrap(t *testing.T) {
	m := testModel()
	m.Visible = true
	require.True(t, m.getWrap())
	cmd := m.Update(key("w"))
	require.False(t, m.getWrap())
	require.NotNil(t, cmd) // WrapToggledMsg
}

func TestKey_Q_Disabled(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("q"))
	require.True(t, m.Visible) // q no longer closes view
}

func TestKey_Esc_ClosesView(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("esc"))
	require.False(t, m.Visible)
}

// --- Key handling: search mode ---

func TestSearchMode_TypeAndConfirm(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.lines = []string{"hello world", "foo bar"}
	m.mu.Unlock()
	m.Update(key("ctrl+f"))
	require.Equal(t, "search", m.mode)
	m.Update(key("h"))
	m.Update(key("e"))
	require.Equal(t, "he", m.searchTerm)
	m.Update(key("enter"))
	require.Equal(t, "normal", m.mode)
	require.Equal(t, "he", m.searchTerm) // preserved after confirm
}

func TestSearchMode_EscCancels(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("ctrl+f"))
	m.Update(key("t"))
	m.Update(key("esc"))
	require.Equal(t, "normal", m.mode)
}

func TestSearchMode_Backspace(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("ctrl+f"))
	m.Update(key("a"))
	m.Update(key("b"))
	require.Equal(t, "ab", m.searchTerm)
	m.Update(key("backspace"))
	require.Equal(t, "a", m.searchTerm)
}

func TestSearchMode_Navigate_N(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.lines = []string{"aaa", "bbb", "aaa"}
	m.mu.Unlock()
	m.searchTerm = "aaa"
	m.highlightContent()
	require.Len(t, m.searchMatches, 2)
	m.Update(key("n"))
	require.Equal(t, 1, m.searchIndex)
	m.Update(key("N"))
	require.Equal(t, 0, m.searchIndex)
}

// --- Key handling: node select dialog ---

func TestNodeSelect_Navigation(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.nodeSelectVisible = true
	m.nodeSelectNodes = []string{"All nodes", "node1", "node2"}
	m.nodeSelectCursor = 0
	m.mu.Unlock()

	m.Update(key("down"))
	m.mu.Lock()
	require.Equal(t, 1, m.nodeSelectCursor)
	m.mu.Unlock()

	m.Update(key("up"))
	m.mu.Lock()
	require.Equal(t, 0, m.nodeSelectCursor)
	m.mu.Unlock()
}

func TestNodeSelect_EnterSelects(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.nodeSelectVisible = true
	m.nodeSelectNodes = []string{"All nodes", "node1"}
	m.nodeSelectCursor = 1
	m.mu.Unlock()

	cmd := m.Update(key("enter"))
	require.False(t, m.GetNodeSelectVisible())
	require.Equal(t, "node1", m.getNodeFilter())
	require.NotNil(t, cmd) // NodeFilterToggledMsg
}

func TestNodeSelect_EnterAllNodes(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.setNodeFilter("node1")
	m.mu.Lock()
	m.nodeSelectVisible = true
	m.nodeSelectNodes = []string{"All nodes", "node1"}
	m.nodeSelectCursor = 0
	m.mu.Unlock()

	m.Update(key("enter"))
	require.Equal(t, "", m.getNodeFilter())
}

func TestNodeSelect_EscCloses(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.nodeSelectVisible = true
	m.nodeSelectNodes = []string{"All nodes"}
	m.mu.Unlock()

	m.Update(key("esc"))
	require.False(t, m.GetNodeSelectVisible())
}

// --- View tests ---

func TestView_NotVisible(t *testing.T) {
	m := testModel()
	m.Visible = false
	out := m.View()
	require.Empty(t, out)
}

func TestView_Normal(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	out := ansi.Strip(m.View())
	require.Contains(t, out, "Logs(web)[0]")
	require.Contains(t, out, "Autoscroll:On")
	require.Contains(t, out, "Wrap:On")
	require.Contains(t, out, "Node:all")
	require.Contains(t, out, "Stopped:Hidden")
}

func TestView_SearchMode(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	m.mode = "search"
	m.searchTerm = "test"
	out := ansi.Strip(m.View())
	require.Contains(t, out, "Search:test_")
}

func TestView_NodeFilter(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	m.setNodeFilter("worker1")
	out := ansi.Strip(m.View())
	require.Contains(t, out, "Node:worker1")
}

// --- frame title / header ---

func TestFrameTitle_ScopesAndCounts(t *testing.T) {
	m := testModel()
	m.ServiceEntry.StackName = "postgres-ha"
	m.SetContent("one\ntwo\nthree")

	require.Equal(t, "Logs(postgres-ha/web)[3]", ansi.Strip(m.FrameTitle()))

	// A service in no stack keeps the scope to the service name.
	m.ServiceEntry.StackName = ""
	require.Equal(t, "Logs(web)[3]", ansi.Strip(m.FrameTitle()))
}

func TestFrameTitle_CountIsPostFilter(t *testing.T) {
	m := hideStoppedModel()
	m.mu.Lock()
	m.lines = []string{"running line", "stopped line"}
	m.lineTasks = []string{"task-run", "task-stop"}
	m.lineNodes = []string{"n", "n"}
	m.mu.Unlock()
	m.viewport.SetContent(m.buildContent())

	require.Equal(t, "Logs(web)[1]", ansi.Strip(m.FrameTitle()), "the hidden stopped line is not counted")

	m.setHideStopped(false)
	m.viewport.SetContent(m.buildContent())
	require.Equal(t, "Logs(web)[2]", ansi.Strip(m.FrameTitle()))
}

func TestFrameTitle_ShowsTheAppFilter(t *testing.T) {
	m := testModel()
	m.SetContent("alpha\nbeta")
	m.ApplySearchQuery("alp")

	require.Equal(t, "Logs(web)[1] </alp>", ansi.Strip(m.FrameTitle()))
}

func TestFrameHeader_ReflectsEveryToggle(t *testing.T) {
	trueColour(t)

	m := testModel()
	m.Visible = true
	m.viewport.Width = 100

	require.Equal(t,
		[]string{"Autoscroll:On", "Wrap:On", "Node:all", "Stopped:Hidden"},
		headerItems(m))

	m.Update(key("s"))
	m.Update(key("w"))
	m.Update(key("t"))
	m.setNodeFilter("worker1")

	require.Equal(t,
		[]string{"Autoscroll:Off", "Wrap:Off", "Node:worker1", "Stopped:Shown"},
		headerItems(m))
}

func TestFrameHeader_ShowsSearchState(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 100
	require.NotContains(t, ansi.Strip(m.FrameHeader()), "Search", "no search running, no item")

	// Typing: the term so far, with a caret marking the field as live.
	m.Update(key("ctrl+f"))
	require.Contains(t, headerItems(m), "Search:_")
	m.Update(key("e"))
	require.Contains(t, headerItems(m), "Search:e_")

	// Confirmed: the match counter the header used to spell out in prose.
	m.mu.Lock()
	m.lines = []string{"error one", "fine", "error two"}
	m.lineNodes = []string{"", "", ""}
	m.lineTasks = []string{"", "", ""}
	m.mu.Unlock()
	m.searchTerm = "error"
	m.Update(key("enter"))
	require.Contains(t, headerItems(m), "Search:error(1/2)")

	m.Update(key("n"))
	require.Contains(t, headerItems(m), "Search:error(2/2)")

	// No matches, and a term long enough to be capped — an uncapped one would
	// push the item past the row's width and lose it altogether.
	m.searchTerm = "nothing-matches-at-all"
	m.highlightContent()
	require.Contains(t, headerItems(m), "Search:nothing-matches…(0)")
}

// TestFrameHeader_IsAlwaysOneRow — views/logs/update.go sizes the viewport to
// what is left after the header, counting its rows. A header that wraps would
// silently take rows off the log content.
func TestFrameHeader_IsAlwaysOneRow(t *testing.T) {
	trueColour(t)

	m := testModel()
	m.setNodeFilter("a-node-with-a-rather-long-hostname")
	m.mode = "search"
	m.searchTerm = strings.Repeat("x", 60)

	for _, width := range []int{10, 20, 40, 80, 120, 200} {
		m.viewport.Width = width
		header := m.FrameHeader()
		require.NotContains(t, header, "\n", "width %d", width)
		require.LessOrEqual(t, lipgloss.Width(header), width, "width %d", width)
	}
}

// headerItems is the frame header split back into its items, so a test names
// the item it cares about instead of the spacing between them.
func headerItems(m *Model) []string {
	return strings.Fields(ansi.Strip(m.FrameHeader()))
}

// trueColour makes a test's assertions run against the styled header. The
// default profile under `go test` is Ascii, where every style renders as plain
// text — so nothing would exercise the ANSI-aware widths the row is built on.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

// --- buildContent tests ---

func TestBuildContent_NodeFilter(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"line1", "line2", "line3"}
	m.lineNodes = []string{"node1", "node2", "node1"}
	m.mu.Unlock()
	m.setNodeFilter("node1")
	content := m.buildContent()
	require.Contains(t, content, "line1")
	require.NotContains(t, content, "line2")
	require.Contains(t, content, "line3")
}

func TestBuildContent_NoFilter(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"line1", "line2"}
	m.lineNodes = []string{"node1", "node2"}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "line1")
	require.Contains(t, content, "line2")
}

func TestSetContent(t *testing.T) {
	m := testModel()
	m.SetContent("a\nb\nc")
	m.mu.Lock()
	require.Len(t, m.lines, 3)
	m.mu.Unlock()
}

func TestSetContent_MaxLines(t *testing.T) {
	m := testModel()
	m.MaxLines = 2
	m.SetContent("a\nb\nc\nd")
	m.mu.Lock()
	require.Len(t, m.lines, 2)
	require.Equal(t, "c", m.lines[0])
	require.Equal(t, "d", m.lines[1])
	m.mu.Unlock()
}

// --- shouldFallbackToRawFromStdCopy tests ---

// --- Filterable (app-level "/" filter) tests ---

func TestApplySearchQuery(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"hello world", "foo bar", "hello again"}
	m.mu.Unlock()
	m.ApplySearchQuery("hello")
	require.Equal(t, "hello", m.getFilterQuery())
	content := m.buildContent()
	require.Contains(t, content, "hello world")
	require.NotContains(t, content, "foo bar")
	require.Contains(t, content, "hello again")
}

func TestClearSearchQuery(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"hello world", "foo bar"}
	m.mu.Unlock()
	m.ApplySearchQuery("hello")
	require.Equal(t, "hello", m.getFilterQuery())
	m.ClearSearchQuery()
	require.Equal(t, "", m.getFilterQuery())
	content := m.buildContent()
	require.Contains(t, content, "hello world")
	require.Contains(t, content, "foo bar")
}

func TestHasActiveFilter_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.HasActiveFilter())
}

func TestHasActiveFilter_WithQuery(t *testing.T) {
	m := testModel()
	m.ApplySearchQuery("test")
	require.True(t, m.HasActiveFilter())
}

func TestCapturesInput(t *testing.T) {
	m := testModel()
	require.False(t, m.CapturesInput())
	m.setNodeSelectVisible(true)
	require.True(t, m.CapturesInput())
}

func TestBuildContent_WithFilterQuery(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"error: something", "info: all good", "error: another"}
	m.mu.Unlock()
	m.mu.Lock()
	m.filterQuery = "error"
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "error: something")
	require.NotContains(t, content, "info: all good")
	require.Contains(t, content, "error: another")
}

func TestFilterAndNodeFilter_Combined(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lines = []string{"error on node1", "info on node1", "error on node2", "info on node2"}
	m.lineNodes = []string{"node1", "node1", "node2", "node2"}
	m.mu.Unlock()
	m.setNodeFilter("node1")
	m.mu.Lock()
	m.filterQuery = "error"
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "error on node1")
	require.NotContains(t, content, "info on node1")
	require.NotContains(t, content, "error on node2")
	require.NotContains(t, content, "info on node2")
}

func TestEsc_ClearsFilterBeforeClosing(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.mu.Lock()
	m.filterQuery = "test"
	m.mu.Unlock()
	m.Update(key("esc"))
	// Filter should be cleared but view stays visible
	require.True(t, m.Visible)
	require.Equal(t, "", m.getFilterQuery())
}

func TestEsc_ClosesViewWhenNoFilter(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("esc"))
	require.False(t, m.Visible)
}

func TestLineMsg_StripsCR(t *testing.T) {
	m := testModel()
	m.Visible = true
	lines := make(chan string, 10)
	errs := make(chan error, 1)
	m.linesChan = lines
	m.errChan = errs

	m.Update(LineMsg{Line: "node1\x00task-1\x00downloading 50%\rprogress"})

	m.mu.Lock()
	require.Len(t, m.lines, 1)
	require.Equal(t, "downloading 50%progress", m.lines[0])
	require.NotContains(t, m.lines[0], "\r")
	m.mu.Unlock()
	close(lines)
	close(errs)
}

func TestSetContent_StripsCR(t *testing.T) {
	m := testModel()
	m.SetContent("line1\r\nline2\r\nline3")
	m.mu.Lock()
	for _, line := range m.lines {
		require.NotContains(t, line, "\r")
	}
	require.Len(t, m.lines, 3)
	require.Equal(t, "line1", m.lines[0])
	require.Equal(t, "line2", m.lines[1])
	require.Equal(t, "line3", m.lines[2])
	m.mu.Unlock()
}

func TestBuildContent_NoWrap_ANSIAware(t *testing.T) {
	m := testModel()
	m.viewport.Width = 20
	m.setWrap(false)

	// Line with ANSI color codes (like formatLogLineWithNode produces)
	ansiLine := "\033[38;5;117mweb.task@node\033[0m | hello world message here"
	m.mu.Lock()
	m.lines = []string{ansiLine}
	m.lineNodes = []string{"node1"}
	m.mu.Unlock()

	content := m.buildContent()

	// Content should not contain broken ANSI sequences (partial escape without 'm')
	// A broken sequence would look like \033[38;5; without the closing digit+m
	require.NotContains(t, content, "\033[38;5;1>")
	require.NotContains(t, content, "\033[38;5;11>")
	// The truncated content should end with > indicator since the line is long
	require.Contains(t, content, ">")
}

func TestShouldFallbackToRawFromStdCopy(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"tty error", errors.New("tty service logs only supported with --raw"), true},
		{"tty error mixed case", errors.New("TTY Service Logs Only Supported With --raw"), true},
		{"unrecognized header", errors.New("unrecognized input header"), true},
		{"short write is not a fallback", io.ErrShortWrite, false},
		{"context canceled", errors.New("context canceled"), false},
		{"random error", errors.New("something went wrong"), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, shouldFallbackToRawFromStdCopy(tc.err))
		})
	}
}

// --- hide-stopped (issue #388) tests ---

// hideStoppedModel returns a model whose snapshot has one running, one starting
// and one stopped (terminal) task for the model's service (svc-123).
func hideStoppedModel() *Model {
	m := testModel()
	m.deps.Snapshot = &mockSnapshotOps{snap: &docker.SwarmSnapshot{
		Tasks: []swarm.Task{
			{ID: "task-run", ServiceID: "svc-123", Status: swarm.TaskStatus{State: swarm.TaskStateRunning}},
			{ID: "task-start", ServiceID: "svc-123", Status: swarm.TaskStatus{State: swarm.TaskStateStarting}},
			{ID: "task-stop", ServiceID: "svc-123", Status: swarm.TaskStatus{State: swarm.TaskStateShutdown}},
		},
	}}
	return m
}

func TestHideStopped_Default(t *testing.T) {
	m := testModel()
	require.True(t, m.getHideStopped())
}

func TestHideStoppedToggle(t *testing.T) {
	m := testModel()
	m.setHideStopped(false)
	require.False(t, m.getHideStopped())
	m.setHideStopped(true)
	require.True(t, m.getHideStopped())
}

func TestKey_T_TogglesHideStopped(t *testing.T) {
	m := testModel()
	m.Visible = true
	require.True(t, m.getHideStopped())
	cmd := m.Update(key("t"))
	require.False(t, m.getHideStopped())
	require.NotNil(t, cmd) // HideStoppedToggledMsg
	m.Update(key("t"))
	require.True(t, m.getHideStopped())
}

func TestBuildContent_HidesStoppedTasks(t *testing.T) {
	m := hideStoppedModel()
	m.mu.Lock()
	m.lines = []string{"running line", "stopped line", "system line"}
	m.lineTasks = []string{"task-run", "task-stop", ""}
	m.lineNodes = []string{"n", "n", ""}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "running line")
	require.NotContains(t, content, "stopped line")
	require.Contains(t, content, "system line") // empty task ID => always visible
}

// TestBuildContent_ShowsStartingAndUnknownTasks is the regression guard for
// issue #388 comment 4644247107: starting containers (non-terminal state) and
// tasks not yet present in the snapshot must remain visible while hide-stopped
// is on.
func TestBuildContent_ShowsStartingAndUnknownTasks(t *testing.T) {
	m := hideStoppedModel()
	require.True(t, m.getHideStopped())
	m.mu.Lock()
	m.lines = []string{"starting line", "unknown line", "stopped line"}
	m.lineTasks = []string{"task-start", "task-not-in-snapshot", "task-stop"}
	m.lineNodes = []string{"n", "n", "n"}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "starting line")   // starting => not terminal => visible
	require.Contains(t, content, "unknown line")    // not in snapshot => fail open => visible
	require.NotContains(t, content, "stopped line") // terminal => hidden
}

func TestBuildContent_ShowAllWhenHideStoppedOff(t *testing.T) {
	m := hideStoppedModel()
	m.setHideStopped(false)
	m.mu.Lock()
	m.lines = []string{"running line", "stopped line", "system line"}
	m.lineTasks = []string{"task-run", "task-stop", ""}
	m.lineNodes = []string{"n", "n", ""}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "running line")
	require.Contains(t, content, "stopped line")
	require.Contains(t, content, "system line")
}

func TestBuildContent_NilSnapshotFailOpen(t *testing.T) {
	m := testModel() // default mock => nil snapshot
	require.True(t, m.getHideStopped())
	m.mu.Lock()
	m.lines = []string{"a", "b"}
	m.lineTasks = []string{"task-run", "task-stop"}
	m.lineNodes = []string{"", ""}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "a")
	require.Contains(t, content, "b") // nil snapshot => show all
}

func TestBuildContent_EmptyTaskIDAlwaysVisible(t *testing.T) {
	m := hideStoppedModel()
	m.mu.Lock()
	m.lines = []string{"no task line"}
	m.lineTasks = []string{""}
	m.lineNodes = []string{""}
	m.mu.Unlock()
	content := m.buildContent()
	require.Contains(t, content, "no task line")
}

func TestHighlightContent_SyncWithHideStopped(t *testing.T) {
	m := hideStoppedModel()
	m.mu.Lock()
	m.lines = []string{"match running", "match stopped", "match again"}
	m.lineTasks = []string{"task-run", "task-stop", "task-run"}
	m.lineNodes = []string{"n", "n", "n"}
	m.mu.Unlock()
	m.searchTerm = "match"
	m.highlightContent()
	// The stopped line is hidden, so only two visible matches remain and their
	// visible indices are contiguous (0,1) — aligned with buildContent's output.
	require.Equal(t, []int{0, 1}, m.searchMatches)
}

func TestSetContent_ResetsTaskSlice(t *testing.T) {
	m := testModel()
	m.mu.Lock()
	m.lineTasks = []string{"old"}
	m.lineNodes = []string{"old"}
	m.mu.Unlock()
	m.SetContent("a\nb\nc")
	m.mu.Lock()
	require.Len(t, m.lineTasks, 3)
	require.Len(t, m.lineNodes, 3)
	for _, id := range m.lineTasks {
		require.Equal(t, "", id)
	}
	m.mu.Unlock()
}

// TestFollowShowsTheNewestLine — the viewport used to be sized to the whole
// frame while the frame drew fewer rows than that, so AutoScroll parked the
// newest lines in the rows that were cut off.
func TestFollowShowsTheNewestLine(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.True(t, m.getFollow())

	for i := 0; i < 500; i++ {
		m.Update(LineMsg{Line: fmt.Sprintf("line-%03d", i)})
	}

	rows := strings.Split(m.FrameContent(), "\n")
	// 40 rows of frame less its two borders and the one-line header.
	require.Len(t, rows, 37)
	require.Contains(t, rows[len(rows)-1], "line-499")
}

// TestResizeKeepsFollowOnTheNewestLine — the viewport kept the offset it had
// across a resize, so "f" grew it into rows it then left blank, and leaving
// fullscreen shrank it away from the newest lines. Both until the next line
// arrived or a keypress clamped the offset.
func TestResizeKeepsFollowOnTheNewestLine(t *testing.T) {
	cases := []struct {
		name           string
		from, to, rows int
	}{
		{name: "growing into fullscreen", from: 24, to: 40, rows: 37},
		{name: "shrinking back to normal", from: 40, to: 24, rows: 21},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			m.Update(tea.WindowSizeMsg{Width: 120, Height: tc.from})
			for i := 0; i < 500; i++ {
				m.Update(LineMsg{Line: fmt.Sprintf("line-%03d", i)})
			}

			m.Update(tea.WindowSizeMsg{Width: 120, Height: tc.to})

			rows := strings.Split(m.FrameContent(), "\n")
			require.Len(t, rows, tc.rows)
			require.Contains(t, rows[len(rows)-1], "line-499")
		})
	}
}

// TestResizeWithFollowOffClampsWithoutJumping — a resize must not drag a reader
// who scrolled up back to the newest line, but it must still pull in an offset
// the taller viewport has left past the end.
func TestResizeWithFollowOffClampsWithoutJumping(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	for i := 0; i < 500; i++ {
		m.Update(LineMsg{Line: fmt.Sprintf("line-%03d", i)})
	}
	m.setFollow(false)

	m.viewport.SetYOffset(100)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 100, m.viewport.YOffset)

	m.viewport.GotoBottom()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	rows := strings.Split(m.FrameContent(), "\n")
	require.Len(t, rows, 57)
	require.Contains(t, rows[len(rows)-1], "line-499")
}

// search returns the matches after typing term and pressing enter, the way a
// user reaches them.
func search(m *Model, term string) {
	m.mode = "search"
	for _, r := range term {
		HandleKey(m, key(string(r)))
	}
	HandleKey(m, key("enter"))
}

// TestSearchCountsLinesThatArriveLater — the incremental update was guarded on
// "no filter is switched on", and hide-stopped is on by default, so on a live
// stream the counter froze at whatever the search found when it was entered
// and n/N navigated a stale set (#586).
func TestSearchCountsLinesThatArriveLater(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(m *Model)
	}{
		{"hide-stopped, the default", func(*Model) {}},
		{"node filter", func(m *Model) { m.setNodeFilter("node-a") }},
		{"slash filter", func(m *Model) { m.ApplySearchQuery("needle") }},
		{"no filter at all", func(m *Model) { m.setHideStopped(false) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := hideStoppedModel()
			m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
			tc.setup(m)

			m.Update(LineMsg{Line: "node-a\x00task-run\x00first needle"})
			search(m, "needle")
			require.Len(t, m.searchMatches, 1)

			for i := 0; i < 3; i++ {
				m.Update(LineMsg{Line: "node-a\x00task-run\x00another needle"})
			}
			require.Equal(t, []int{0, 1, 2, 3}, m.searchMatches,
				"matches in lines that arrived after the search must be counted")
		})
	}
}

// TestSearchMatchesAreVisibleIndices — a hidden line must not advance the index
// the matches are numbered by, or n/N scrolls to the wrong row.
func TestSearchMatchesAreVisibleIndices(t *testing.T) {
	m := hideStoppedModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})

	m.Update(LineMsg{Line: "node-a\x00task-run\x00needle one"})
	m.Update(LineMsg{Line: "node-a\x00task-stop\x00needle hidden"})
	m.Update(LineMsg{Line: "node-a\x00task-run\x00needle two"})
	search(m, "needle")

	// The shutdown task's line is filtered out, so the two visible matches are
	// numbered 0 and 1 rather than 0 and 2.
	require.Equal(t, []int{0, 1}, m.searchMatches)
	require.Equal(t, 2, m.getVisibleCount())
}

// TestSearchSurvivesTheBufferWrapping — once MaxLines is reached every new line
// drops one off the top, which used to discard the whole match set on each
// line and leave the counter reading zero for the rest of the session.
func TestSearchSurvivesTheBufferWrapping(t *testing.T) {
	m := hideStoppedModel()
	m.MaxLines = 4
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})

	m.Update(LineMsg{Line: "node-a\x00task-run\x00needle"})
	search(m, "needle")
	require.Len(t, m.searchMatches, 1)

	// Six more lines through a four-line buffer: the first needle is long gone
	// and the three that remain are the last three appended.
	for i := 0; i < 3; i++ {
		m.Update(LineMsg{Line: "node-a\x00task-run\x00filler"})
	}
	for i := 0; i < 3; i++ {
		m.Update(LineMsg{Line: "node-a\x00task-run\x00needle"})
	}
	require.Equal(t, []int{1, 2, 3}, m.searchMatches)
}
