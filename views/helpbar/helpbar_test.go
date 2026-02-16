package helpbar

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNew_DefaultGlobalHelp(t *testing.T) {
	m := New(200, 40)
	require.Len(t, m.globalHelp, 2)
	require.Equal(t, "q", m.globalHelp[0].Key)
	require.Equal(t, "?", m.globalHelp[1].Key)
}

func TestView_WithEntries(t *testing.T) {
	m := New(200, 40)
	m.WithViewHelp([]HelpEntry{
		{Key: "n", Desc: "New"},
		{Key: "d", Desc: "Delete"},
	})
	out := m.View("", false)
	require.Contains(t, out, "q")
	require.Contains(t, out, "help")
	require.Contains(t, out, "n")
	require.Contains(t, out, "New")
	require.Contains(t, out, "_____") // ASCII art logo present
}

func TestView_DisabledEntryNotBold(t *testing.T) {
	m := New(200, 40)
	m.WithViewHelp([]HelpEntry{
		{Key: "x", Desc: "Reveal (Pro)", Disabled: true},
	})
	out := m.View("", false)
	// Disabled entries still appear in output, just styled differently
	require.Contains(t, out, "x")
	require.Contains(t, out, "Reveal (Pro)")
}

func TestView_EmptyHelp_ReturnsSystemInfo(t *testing.T) {
	m := New(200, 40)
	m.globalHelp = nil
	m.viewHelp = nil
	out := m.View("sys-info-block", false)
	require.Equal(t, "sys-info-block", out)
}

func TestView_HasError_StillRendersLogo(t *testing.T) {
	m := New(200, 40)
	out := m.View("", true)
	// Logo is rendered even in error mode (color differs at runtime with terminal)
	require.Contains(t, out, "_____")
}

func TestView_NarrowWidth_SkipsHelp(t *testing.T) {
	m := New(30, 40) // Very narrow
	out := m.View("wide-system-info-panel-here!!", false)
	// Not enough space for help columns; should just return systemInfo
	require.Equal(t, "wide-system-info-panel-here!!", out)
}

func TestSetters(t *testing.T) {
	m := New(80, 24)
	m.SetWidth(100)
	require.Equal(t, 100, m.width)
	m.SetHeight(50)
	require.Equal(t, 50, m.height)
	m.SetMinColWidth(30)
	require.Equal(t, 30, m.minColWidth)
}

func TestWithGlobalHelp(t *testing.T) {
	m := New(200, 40)
	custom := []HelpEntry{{Key: "a", Desc: "action"}}
	m.WithGlobalHelp(custom)
	require.Equal(t, custom, m.globalHelp)
}
