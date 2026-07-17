// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The regression this feature exists for: `--set 'extraNetworks[0]=x'` used to be
// accepted and then silently do nothing — it landed as a key literally named
// "extraNetworks[0]" that no template reads, so the install succeeded and the
// operator debugged the wrong thing. It must now actually build the list.
func TestMergeValuesSetListIndex(t *testing.T) {
	got, err := MergeValues(map[string]any{}, nil, []string{
		"extraNetworks[0]=ollama-net",
		"extraNetworks[1]=mail-net",
	})
	require.NoError(t, err)
	require.Equal(t, []any{"ollama-net", "mail-net"}, got["extraNetworks"])
	require.NotContains(t, got, "extraNetworks[0]", "must not leak a bracketed key")
}

// Helm's list literal, the ergonomic form for a list of scalars.
func TestMergeValuesSetListLiteral(t *testing.T) {
	got, err := MergeValues(map[string]any{}, nil, []string{"extraNetworks={ollama-net,mail-net}"})
	require.NoError(t, err)
	require.Equal(t, []any{"ollama-net", "mail-net"}, got["extraNetworks"])

	// A comma inside the expression must not be mistaken for a --set separator.
	got, err = MergeValues(map[string]any{}, nil, []string{"a={x,y},b=1"})
	require.NoError(t, err)
	require.Equal(t, []any{"x", "y"}, got["a"])
	require.Equal(t, 1, got["b"])

	empty, err := MergeValues(map[string]any{}, nil, []string{"a={}"})
	require.NoError(t, err)
	require.Equal(t, []any{}, empty["a"])
}

// Elements are type-inferred, and a list replaces a default rather than merging.
func TestMergeValuesSetListReplacesDefault(t *testing.T) {
	defaults := map[string]any{"extraNetworks": []any{"stale-net", "other"}}
	got, err := MergeValues(defaults, nil, []string{"extraNetworks={only-net}"})
	require.NoError(t, err)
	require.Equal(t, []any{"only-net"}, got["extraNetworks"])
	require.Equal(t, []any{"stale-net", "other"}, defaults["extraNetworks"], "defaults must not be mutated")

	mixed, err := MergeValues(map[string]any{}, nil, []string{"a={1,true,x}"})
	require.NoError(t, err)
	require.Equal(t, []any{1, true, "x"}, mixed["a"])
}

// Indexing into a list of maps, Helm-style.
func TestMergeValuesSetListOfMaps(t *testing.T) {
	got, err := MergeValues(map[string]any{}, nil, []string{"servers[0].port=80", "servers[0].host=a", "servers[1].port=443"})
	require.NoError(t, err)
	require.Equal(t, []any{
		map[string]any{"port": 80, "host": "a"},
		map[string]any{"port": 443},
	}, got["servers"])
}

// A sparse index pads with nil rather than failing.
func TestMergeValuesSetListSparseIndex(t *testing.T) {
	got, err := MergeValues(map[string]any{}, nil, []string{"a[2]=x"})
	require.NoError(t, err)
	require.Equal(t, []any{nil, nil, "x"}, got["a"])
}

// Escaped commas survive both the --set split and the list split.
func TestMergeValuesSetEscapedCommas(t *testing.T) {
	got, err := MergeValues(map[string]any{}, nil, []string{`a=x\,y`})
	require.NoError(t, err)
	require.Equal(t, "x,y", got["a"])

	lst, err := MergeValues(map[string]any{}, nil, []string{`a={x\,y,z}`})
	require.NoError(t, err)
	require.Equal(t, []any{"x,y", "z"}, lst["a"])
}

// A malformed path must fail loudly — never land as an unread bracketed key.
func TestMergeValuesSetListErrors(t *testing.T) {
	for _, expr := range []string{
		"a[x]=1",  // non-numeric index
		"a[-1]=1", // negative index
		"a[0=1",   // unbalanced '['
		"a]0]=1",  // unbalanced ']'
		"[0]=1",   // index with no key
		"a..b=1",  // empty segment
		"a[0]x=1", // trailing junk after an index
	} {
		_, err := MergeValues(map[string]any{}, nil, []string{expr})
		require.Error(t, err, "--set %q must be rejected, not silently ignored", expr)
		require.Contains(t, err.Error(), "--set")
	}
}
