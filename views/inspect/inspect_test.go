package inspectview

import (
	"strings"
	"testing"

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

func testModel() *Model {
	return New(80, 24, FormatYAML)
}

const testJSON = `{"name":"test","count":42,"nested":{"key":"value"}}`

// --- Tests ---

func TestNew(t *testing.T) {
	m := New(80, 24, FormatYAML)
	require.Equal(t, FormatYAML, m.Format)
	require.Equal(t, 80, m.width)
	require.Equal(t, 24, m.height)
}

func TestName(t *testing.T) {
	m := testModel()
	require.Equal(t, "inspect", m.Name())
}

func TestHasErrors(t *testing.T) {
	m := testModel()
	require.False(t, m.HasErrors())
}

func TestSetTitle(t *testing.T) {
	m := testModel()
	m.SetTitle("My Title")
	require.Equal(t, "My Title", m.Title)
}

func TestSetFormat(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.SetFormat(FormatRaw)
	require.Equal(t, FormatRaw, m.Format)
	m.SetFormat(FormatYAML)
	require.Equal(t, FormatYAML, m.Format)
}

func TestSetContent_YAML(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	require.NotNil(t, m.Root)
	require.Equal(t, testJSON, m.RawContent)
}

func TestSetContent_Raw(t *testing.T) {
	m := New(80, 24, FormatRaw)
	m.SetContent(testJSON)
	require.Nil(t, m.Root) // raw mode doesn't parse
	require.Equal(t, testJSON, m.RawContent)
}

func TestSetContent_InvalidJSON(t *testing.T) {
	m := testModel()
	m.SetContent("not json")
	require.NotEmpty(t, m.ParseError)
	require.Equal(t, FormatRaw, m.Format) // fallback
}

func TestParseFormat(t *testing.T) {
	require.Equal(t, FormatYAML, ParseFormat("yml"))
	require.Equal(t, FormatRaw, ParseFormat("raw"))
	require.Equal(t, FormatYAML, ParseFormat("unknown"))
	require.Equal(t, FormatYAML, ParseFormat(FormatYAML))
	require.Equal(t, FormatRaw, ParseFormat(FormatRaw))
}

func TestShortHelpItems_Normal(t *testing.T) {
	m := testModel()
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["ctrl+f"])
	require.True(t, keys["r"])
	require.True(t, keys["q"])
}

func TestShortHelpItems_SearchMode(t *testing.T) {
	m := testModel()
	m.searchMode = true
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["enter"])
	require.True(t, keys["esc"])
}

func TestLoadInspectItem(t *testing.T) {
	cmd := LoadInspectItem("Title", testJSON)
	msg := cmd()
	inspectMsg, ok := msg.(Msg)
	require.True(t, ok)
	require.Equal(t, "Title", inspectMsg.Title)
	require.Equal(t, testJSON, inspectMsg.Content)
}

// --- Update tests ---

func TestUpdate_Msg(t *testing.T) {
	m := testModel()
	m.Update(Msg{Title: "Test", Content: testJSON})
	require.Equal(t, "Test", m.Title)
	require.True(t, m.ready)
	require.NotNil(t, m.Root)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.viewport.Width)
	require.Equal(t, 40, m.viewport.Height)
	require.True(t, m.ready)
}

// --- Key handling: normal mode ---

func TestKey_R_ToggleFormat(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.ready = true
	require.Equal(t, FormatYAML, m.Format)
	m.Update(key("r"))
	require.Equal(t, FormatRaw, m.Format)
	m.Update(key("r"))
	require.Equal(t, FormatYAML, m.Format)
}

func TestKey_CtrlF_EntersSearch(t *testing.T) {
	m := testModel()
	m.ready = true
	m.Update(key("ctrl+f"))
	require.True(t, m.searchMode)
	require.Equal(t, "", m.SearchTerm)
}

// --- Key handling: search mode ---

func TestSearch_TypeAndConfirm(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.ready = true
	m.Update(key("ctrl+f"))
	m.Update(key("n"))
	m.Update(key("a"))
	require.Equal(t, "na", m.SearchTerm)
	m.Update(key("enter"))
	require.False(t, m.searchMode)
	require.Equal(t, "na", m.SearchTerm) // preserved
}

func TestSearch_EscCancels(t *testing.T) {
	m := testModel()
	m.ready = true
	m.Update(key("ctrl+f"))
	m.Update(key("x"))
	m.Update(key("esc"))
	require.False(t, m.searchMode)
	require.Equal(t, "", m.SearchTerm)
}

func TestSearch_Backspace(t *testing.T) {
	m := testModel()
	m.ready = true
	m.Update(key("ctrl+f"))
	m.Update(key("a"))
	m.Update(key("b"))
	require.Equal(t, "ab", m.SearchTerm)
	m.Update(key("backspace"))
	require.Equal(t, "a", m.SearchTerm)
}

// --- ParseJSON ---

func TestParseJSON_ValidObject(t *testing.T) {
	root, err := ParseJSON(`{"key":"value","num":42}`)
	require.NoError(t, err)
	require.NotNil(t, root)
	require.True(t, len(root.Children) >= 2)
}

func TestParseJSON_NestedObject(t *testing.T) {
	root, err := ParseJSON(`{"outer":{"inner":"val"}}`)
	require.NoError(t, err)
	require.NotNil(t, root)
}

func TestParseJSON_Array(t *testing.T) {
	root, err := ParseJSON(`{"items":[1,2,3]}`)
	require.NoError(t, err)
	require.NotNil(t, root)
}

func TestParseJSON_Invalid(t *testing.T) {
	_, err := ParseJSON("not json")
	require.Error(t, err)
}

// --- View ---

func TestView_YAML(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	out := m.viewport.View()
	require.NotEmpty(t, out)
}

// --- Filterable (app-level "/" filter) tests ---

func TestApplySearchQuery_FiltersLines(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.ready = true
	m.ApplySearchQuery("test")
	// After filtering, only lines containing "test" should remain
	visible := m.visiblePlainLines()
	for _, line := range visible {
		require.Contains(t, strings.ToLower(line), "test")
	}
}

func TestClearSearchQuery_RestoresAll(t *testing.T) {
	m := testModel()
	m.SetContent(testJSON)
	m.ready = true
	allBefore := m.visiblePlainLines()
	m.ApplySearchQuery("test")
	m.ClearSearchQuery()
	allAfter := m.visiblePlainLines()
	require.Equal(t, len(allBefore), len(allAfter))
}

func TestHasActiveFilter_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.HasActiveFilter())
}

func TestIsSearching_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.IsSearching())
}

func TestIsSearching_SearchMode(t *testing.T) {
	m := testModel()
	m.searchMode = true
	require.True(t, m.IsSearching())
}

func TestSearch_NNavigation(t *testing.T) {
	m := testModel()
	m.SetContent(`{"name":"test","count":42,"nested":{"name":"test2"}}`)
	m.ready = true
	m.viewport.Width = 80
	m.viewport.Height = 24
	// Enter search, type "name", confirm
	m.Update(key("ctrl+f"))
	m.Update(key("n"))
	m.Update(key("a"))
	m.Update(key("m"))
	m.Update(key("e"))
	m.Update(key("enter"))
	require.False(t, m.searchMode)
	require.Equal(t, "name", m.SearchTerm)
	require.Greater(t, len(m.searchMatches), 0)

	// Navigate with n
	if len(m.searchMatches) > 1 {
		m.Update(key("n"))
		require.Equal(t, 1, m.searchIndex)
		m.Update(key("N"))
		require.Equal(t, 0, m.searchIndex)
	}
}

func TestEsc_ClearsFilterBeforeGoBack(t *testing.T) {
	m := testModel()
	m.ready = true
	m.filterQuery = "test"
	m.Update(key("esc"))
	// Filter should be cleared
	require.Equal(t, "", m.filterQuery)
	require.False(t, m.HasActiveFilter())
}

func TestApplySearchQuery_RawMode(t *testing.T) {
	m := New(80, 24, FormatRaw)
	m.SetContent("line one\nline two\nline three")
	m.ready = true
	m.ApplySearchQuery("two")
	visible := m.visiblePlainLines()
	require.Len(t, visible, 1)
	require.Contains(t, visible[0], "two")
}
