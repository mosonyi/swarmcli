// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"testing"

	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/ui/dialog"

	"github.com/charmbracelet/x/ansi"

	"github.com/stretchr/testify/require"
)

func TestView_LoadingState_ShowsLoading(t *testing.T) {
	m := testModel()
	m.state = stateLoading
	out := m.View()
	require.Contains(t, out, "Docker Secrets")
}

func TestView_ReadyState_ShowsSecrets(t *testing.T) {
	m := testModel()
	loadSecrets(m, fakeSecrets("alpha", "beta"))
	m.setRenderItem()
	m.secretsList.Viewport.Width = 80
	m.secretsList.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Docker Secrets (2)")
	require.Contains(t, out, "alpha")
	require.Contains(t, out, "beta")
}

func TestView_ErrorDialog_ShowsError(t *testing.T) {
	m := testModel()
	loadSecrets(m, fakeSecrets("s1"))
	m.setRenderItem()
	m.secretsList.Viewport.Width = 80
	m.secretsList.Viewport.Height = 20
	m.errorDialogActive = true
	m.err = errorMsg(errTest)
	out := m.View()
	require.Contains(t, out, "test error")
}

func TestView_CreateDialog_Source(t *testing.T) {
	m := testModel()
	loadSecrets(m, fakeSecrets("s1"))
	m.setRenderItem()
	m.secretsList.Viewport.Width = 80
	m.secretsList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "source"
	out := m.View()
	require.Contains(t, out, "Create Secret")
	require.Contains(t, out, "From file")
}

func TestView_CreateDialog_DetailsFile(t *testing.T) {
	m := testModel()
	loadSecrets(m, fakeSecrets("s1"))
	m.setRenderItem()
	m.secretsList.Viewport.Width = 80
	m.secretsList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	out := m.View()
	require.Contains(t, out, "Create Secret from File")
}

func TestView_UsedByView(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	m.usedBySecretName = "my-secret"
	m.usedByList.Viewport.Width = 80
	m.usedByList.Viewport.Height = 20
	m.usedByList.Items = []usedByItem{
		{StackName: "stack1", ServiceName: "svc1"},
	}
	m.usedByList.Filtered = m.usedByList.Items
	m.usedByList.RenderItem = func(item usedByItem, _ bool, _ int) string {
		return item.StackName + " " + item.ServiceName
	}
	out := m.View()
	require.Contains(t, out, "my-secret")
	require.Contains(t, out, "Used By")
}

func TestFormatLabels_Empty(t *testing.T) {
	require.Equal(t, "-", formatLabels(nil))
	require.Equal(t, "-", formatLabels(map[string]string{}))
}

func TestFormatLabels_SortedOutput(t *testing.T) {
	labels := map[string]string{"b": "2", "a": "1"}
	require.Equal(t, "a=1,b=2", formatLabels(labels))
}

func TestFormatLabels_SkipsInternalLabels(t *testing.T) {
	labels := map[string]string{"swarmcli.internal": "val", "env": "prod"}
	require.Equal(t, "env=prod", formatLabels(labels))
}

func TestLabelsScroll(t *testing.T) {
	full := formatLabels(map[string]string{"longkey": "longvalue"})
	require.Equal(t, "longkey=longvalue", full)

	// Scroll past start (shared rune-aware window).
	require.Equal(t, "key=longvalue", filterlist.ScrollWindow(full, 4, 20))
}

func TestLabelsScroll_Truncation(t *testing.T) {
	full := formatLabels(map[string]string{"a": "verylongvalue"})
	truncated := filterlist.ScrollWindow(full, 0, 5)
	require.Contains(t, truncated, "…")
	require.Equal(t, 5, len([]rune(truncated)))
}

func TestGetSecretsHelpContent(t *testing.T) {
	cats := GetSecretsHelpContent()
	require.True(t, len(cats) >= 3)
	require.Equal(t, "General", cats[0].Title)
	require.Equal(t, "View", cats[1].Title)
	require.Equal(t, "Navigation", cats[2].Title)
}

var errTest = errorMsg(testErr{})

type testErr struct{}

func (testErr) Error() string { return "test error" }

// --- #525: the browse hint advertises the chord, from any focus ---

func TestRenderCreateDialog_ShowsBrowseHintAtEveryFocus(t *testing.T) {
	for _, focus := range []int{0, 1, 2, 3} {
		m := testModel()
		m.createDialogActive = true
		m.createDialogStep = "details-file"
		m.createInputFocus = focus
		out := ansi.Strip(m.renderCreateDialog())
		require.Contains(t, out, dialog.BrowseHint, "focus %d must show the browse hint", focus)
		require.NotContains(t, out, "[f: Browse]")
	}
}
