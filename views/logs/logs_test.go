package logsview

import (
	"errors"
	"io"
	"testing"

	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
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

type mockSnapshotOps struct{}

func (m *mockSnapshotOps) GetSnapshot() *docker.SwarmSnapshot              { return nil }
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
	require.True(t, keys["q"])
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
	m.Update(LineMsg{Line: "node1\x00hello world"})

	m.mu.Lock()
	require.Len(t, m.lines, 1)
	require.Equal(t, "hello world", m.lines[0])
	require.Equal(t, "node1", m.lineNodes[0])
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
	require.Equal(t, 40, m.viewport.Height)
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

func TestKey_Q_ClosesView(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.Update(key("q"))
	require.False(t, m.Visible)
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
	out := m.View()
	require.Contains(t, out, "Service: web")
	require.Contains(t, out, "AutoScroll: on")
	require.Contains(t, out, "wrap: on")
}

func TestView_SearchMode(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	m.mode = "search"
	m.searchTerm = "test"
	out := m.View()
	require.Contains(t, out, "Search: test")
}

func TestView_NodeFilter(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	m.setNodeFilter("worker1")
	out := m.View()
	require.Contains(t, out, "node: worker1")
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

func TestHasActiveDialog(t *testing.T) {
	m := testModel()
	require.False(t, m.HasActiveDialog())
	m.setNodeSelectVisible(true)
	require.True(t, m.HasActiveDialog())
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
