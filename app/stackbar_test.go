// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"strings"
	"testing"

	"swarmcli/views/view"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

// capturingStubView simulates a view that swallows every keystroke, so the
// app-level ":" bar is unreachable from it.
type capturingStubView struct{ stubView }

func (v *capturingStubView) HidesGlobalHelp() bool { return true }

// setStackBarSuffix sets the package-level suffix for the duration of a test.
func setStackBarSuffix(t *testing.T, suffix string) {
	t.Helper()
	orig := StackBarSuffix
	t.Cleanup(func() { StackBarSuffix = orig })
	StackBarSuffix = suffix
}

// deepTrail is a realistic nested trail. Only "stacks" is top-level, so
// RenderBreadcrumbs keeps all five segments rather than trimming to a nearer
// root — this is what exercises the maxDisplay cap and the "…" prefix.
var deepTrail = []string{view.NameStacks, view.NameServices, view.NameTasks, view.NameInspect, view.NameLogs}

// nestedModel returns a model whose view stack renders deepTrail.
func nestedModel(t *testing.T) *Model {
	t.Helper()
	m := newTestAppModel(&stubView{name: deepTrail[len(deepTrail)-1]})
	for _, n := range deepTrail[:len(deepTrail)-1] {
		m.viewStack.Push(&stubView{name: n})
	}
	return m
}

func TestRenderStackBar_ShowsCommandHint(t *testing.T) {
	setStackBarSuffix(t, "")
	m := newTestAppModel(&stubView{name: view.NameStacks})

	out := ansi.Strip(m.renderStackBar())

	require.Contains(t, out, commandHintText)
	require.Contains(t, out, view.NameStacks)
}

func TestRenderStackBar_HintCoexistsWithSuffix(t *testing.T) {
	setStackBarSuffix(t, "status-suffix")
	m := newTestAppModel(&stubView{name: view.NameStacks})

	out := m.renderStackBar()
	stripped := ansi.Strip(out)

	require.Contains(t, stripped, commandHintText)
	require.Contains(t, stripped, "status-suffix")
	// The suffix keeps the right edge; the hint sits to its left.
	require.True(t, strings.HasSuffix(stripped, "status-suffix"),
		"suffix must be flush right, got %q", stripped)
	require.Less(t, strings.Index(stripped, commandHintText), strings.Index(stripped, "status-suffix"))
	require.LessOrEqual(t, lipgloss.Width(out), m.terminalWidth)
}

func TestRenderStackBar_SuffixSurvivesWhenHintDoesNot(t *testing.T) {
	setStackBarSuffix(t, "status-suffix")
	m := newTestAppModel(&stubView{name: view.NameStacks})

	// Exactly enough room for the crumbs and the suffix, but not the hint.
	crumbs := m.fitBreadcrumbs([]string{view.NameStacks})
	m.terminalWidth = lipgloss.Width(crumbs) + stackBarGap + lipgloss.Width(StackBarSuffix)

	out := ansi.Strip(m.renderStackBar())

	require.Contains(t, out, "status-suffix")
	require.NotContains(t, out, commandHintText)
}

func TestRenderStackBar_HiddenInHelpView(t *testing.T) {
	setStackBarSuffix(t, "")
	m := newTestAppModel(&stubView{name: view.NameHelp})

	out := ansi.Strip(m.renderStackBar())

	require.NotContains(t, out, commandHintText)
	require.Contains(t, out, view.NameHelp)
}

func TestRenderStackBar_HiddenWhenViewHidesGlobalHelp(t *testing.T) {
	setStackBarSuffix(t, "")
	m := newTestAppModel(&capturingStubView{stubView{name: "shell"}})

	out := ansi.Strip(m.renderStackBar())

	require.NotContains(t, out, commandHintText)
	require.Contains(t, out, "shell")
}

func TestFitBreadcrumbs_ShedsOldestToFit(t *testing.T) {
	m := nestedModel(t)
	m.terminalWidth = 30

	out := m.fitBreadcrumbs(deepTrail)
	stripped := ansi.Strip(out)

	require.LessOrEqual(t, lipgloss.Width(out), 30)
	require.Contains(t, stripped, "…", "shed segments must collapse into an ellipsis")
	// Shedding is oldest-first: the current view always survives.
	require.Contains(t, stripped, view.NameLogs)
	require.NotContains(t, stripped, view.NameStacks, "oldest segments shed first")
}

func TestFitBreadcrumbs_TruncatesWhenSingleCrumbOverflows(t *testing.T) {
	m := nestedModel(t)
	m.terminalWidth = 5

	out := m.fitBreadcrumbs(deepTrail)

	require.LessOrEqual(t, lipgloss.Width(out), 5)
}

func TestFitBreadcrumbs_UnclampedWhenWidthUnset(t *testing.T) {
	m := nestedModel(t)
	m.terminalWidth = 0

	out := m.fitBreadcrumbs(deepTrail)

	require.Equal(t, RenderBreadcrumbs(deepTrail, stackBarMaxCrumbs), out)
	require.NotEmpty(t, out)
}

func TestRenderStackBar_NeverExceedsTerminalWidth(t *testing.T) {
	orig := StackBarSuffix
	t.Cleanup(func() { StackBarSuffix = orig })

	m := nestedModel(t)

	for w := 10; w <= 200; w += 2 {
		for _, suffix := range []string{"", "status-suffix", strings.Repeat("s", 60)} {
			StackBarSuffix = suffix
			m.terminalWidth = w
			out := m.renderStackBar()
			require.LessOrEqualf(t, lipgloss.Width(out), w, "width overflow at w=%d suffix=%q", w, suffix)
			require.Equalf(t, 1, lipgloss.Height(out), "stack bar must stay one line at w=%d", w)
		}
	}
}
