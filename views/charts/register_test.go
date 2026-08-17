// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

// The payload key is a contract with the stacks view's jump. A typo on either
// side would silently land on an unselected list.
func TestFactoryReadsTheReleasePayload(t *testing.T) {
	v, cmd := factory(docker.Deps{}, 100, 30, map[string]any{"release": "wanted"})
	require.NotNil(t, cmd)

	m, ok := v.(*Model)
	require.True(t, ok)
	require.Equal(t, "wanted", m.pendingSelect)
	require.Equal(t, 100, m.width)
	require.Equal(t, 30, m.height)
}

func TestFactoryToleratesAMissingOrOddPayload(t *testing.T) {
	for name, payload := range map[string]any{
		"nil":          nil,
		"wrong type":   "wanted",
		"missing key":  map[string]any{"stackName": "x"},
		"empty string": map[string]any{"release": ""},
		"not a string": map[string]any{"release": 7},
	} {
		v, _ := factory(docker.Deps{}, 80, 24, payload)
		require.Empty(t, v.(*Model).pendingSelect, "payload %q", name)
	}
}

func TestFactoryIsRegisteredUnderTheViewName(t *testing.T) {
	f, ok := view.GetFactory(ViewName)
	require.True(t, ok, "the view must be reachable by name")
	require.NotNil(t, f)

	v, _ := f(docker.Deps{}, 80, 24, nil)
	require.Equal(t, ViewName, v.Name())
}

// A root view: esc must not pop out of it into whatever came before.
func TestChartsIsATopLevelView(t *testing.T) {
	require.True(t, view.IsTopLevel(ViewName))
	require.Equal(t, view.NameCharts, ViewName)
}

// The helpbar advertises keys the view actually handles.
func TestShortHelpItemsMatchRealKeys(t *testing.T) {
	m := testModel()
	keys := map[string]bool{}
	for _, e := range m.ShortHelpItems() {
		keys[e.Key] = true
		require.NotEmpty(t, e.Desc, "key %q has no description", e.Key)
	}
	for _, want := range []string{"enter", "i", "v", "d", "s", "/", "?"} {
		require.True(t, keys[want], "the helpbar should advertise %q", want)
	}
}
