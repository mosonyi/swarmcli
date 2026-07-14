// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// A chromeless view is rendered verbatim: no help bar, no frame, no breadcrumbs.
func TestChromelessView_RenderedVerbatim(t *testing.T) {
	m := newTestAppModel(&chromelessStubView{})

	out := m.View()

	require.Equal(t, "CHROMELESS", out)
	require.NotContains(t, out, "│")
}

// A chromeless view gets the raw terminal size, while m.viewport stays
// chrome-sized so views further down the stack are restored correctly.
func TestChromelessView_GetsFullTerminalSize(t *testing.T) {
	cv := &chromelessStubView{}
	m := newTestAppModel(cv)

	m.updateForResize(tea.WindowSizeMsg{Width: 100, Height: 40})

	require.Equal(t, []tea.WindowSizeMsg{{Width: 100, Height: 40}}, cv.receivedSizes)
	require.Equal(t, 96, m.viewport.Width)
	require.Equal(t, 40, m.viewport.Height)
}

// A framed view still gets the chrome-reduced size (help bar + breadcrumb bar).
func TestFramedView_GetsChromeReducedSize(t *testing.T) {
	m := newTestAppModel(&stubView{name: "framed"})

	m.updateForResize(tea.WindowSizeMsg{Width: 100, Height: 40})

	require.Equal(t, 96, m.viewport.Width)
	require.Equal(t, 40, m.viewport.Height)
}

// ":" must not open the command bar over a chromeless view — it has nowhere to
// render, and an invisible command bar swallows every subsequent key.
func TestChromelessView_ColonDoesNotOpenCommandInput(t *testing.T) {
	cv := &chromelessStubView{}
	m := newTestAppModel(cv)

	m.Update(runeKey(':'))

	require.False(t, m.commandInput.Visible())
	require.Len(t, cv.receivedKeys, 1)
	require.Equal(t, ":", cv.receivedKeys[0].String())
}

// "/" must not open the search bar over a chromeless view either.
func TestChromelessView_SlashDoesNotOpenSearchInput(t *testing.T) {
	cv := &chromelessStubView{}
	m := newTestAppModel(cv)

	m.Update(runeKey('/'))

	require.False(t, m.searchInput.Visible())
	require.Len(t, cv.receivedKeys, 1)
	require.Equal(t, "/", cv.receivedKeys[0].String())
}

// "f" is not a fullscreen toggle over a chromeless view — it is already
// borderless, so the key belongs to the view.
func TestChromelessView_FDoesNotToggleFullscreen(t *testing.T) {
	cv := &chromelessStubView{}
	m := newTestAppModel(cv)

	m.Update(runeKey('f'))

	require.False(t, m.fullscreen)
	require.Len(t, cv.receivedKeys, 1)
	require.Equal(t, "f", cv.receivedKeys[0].String())
}

// Entering a chromeless view while the app happens to be in fullscreen must not
// cost the user a second Esc: the first Esc goes back, it does not silently
// clear the (invisible) fullscreen flag.
func TestChromelessView_EscGoesBackEvenWhenFullscreen(t *testing.T) {
	cv := &chromelessStubView{}
	m := newTestAppModel(cv)
	m.fullscreen = true
	m.viewStack.Push(&stubView{name: "services"})

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	require.Equal(t, "services", m.currentView.Name())
}
