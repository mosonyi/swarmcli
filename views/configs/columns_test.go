// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

const longConfigName = "a-very-long-config-name-that-overflows-columns"

func readyConfigs(m *Model, width int, names ...string) {
	loadConfigs(m, fakeConfigs(names...))
	m.setRenderItem()
	m.configsList.Viewport.Width = width
	m.configsList.Viewport.Height = 20
}

func TestUsedHeaderIsShortened(t *testing.T) {
	m := testModel()
	readyConfigs(m, 120, "web")
	header := m.configsList.RenderHeader()
	require.Contains(t, header, "USED")
	require.NotContains(t, header, "CONFIG USED")
}

func TestContentAwareName_WideTerminal(t *testing.T) {
	m := testModel()
	readyConfigs(m, 200, longConfigName, "tls")
	row := m.configsList.RenderItem(m.configsList.Filtered[0], false, 0)
	require.Contains(t, row, longConfigName, "full name must be visible on a wide terminal")
}

func TestHeaderRowAlignment(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs(longConfigName, "tls"))
	m.setRenderItem()
	for _, w := range []int{50, 120, 200} {
		m.configsList.Viewport.Width = w
		header := m.configsList.RenderHeader()
		row := m.configsList.RenderItem(m.configsList.Filtered[0], false, 0)
		require.Equal(t, lipgloss.Width(header), lipgloss.Width(row), "width=%d header/row mismatch", w)
	}
}

func TestArrowScrollMovesWindow(t *testing.T) {
	m := testModel()
	readyConfigs(m, 60, longConfigName, "tls")
	before := m.configsList.RenderItem(m.configsList.Filtered[0], true, 0)
	m.Update(key("right"))
	after := m.configsList.RenderItem(m.configsList.Filtered[0], true, 0)
	require.NotEqual(t, before, after, "right arrow should shift the truncated cell window")

	m.Update(key("left"))
	require.Equal(t, before, m.configsList.RenderItem(m.configsList.Filtered[0], true, 0),
		"left arrow should restore the window")
}

func TestResetScrollOnCursorMove(t *testing.T) {
	m := testModel()
	readyConfigs(m, 60, longConfigName, "tls")
	m.Update(key("right"))
	m.Update(key("right"))
	scrolled := m.configsList.RenderItem(m.configsList.Filtered[0], true, 0)

	m.Update(key("down"))
	m.Update(key("up"))

	require.NotEqual(t, scrolled, m.configsList.RenderItem(m.configsList.Filtered[0], true, 0),
		"scroll offset must reset when the cursor moves (configs lacked this before)")
}
