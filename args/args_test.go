// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package args

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGet_ExistingFlag(t *testing.T) {
	a := Args{Flags: map[string]string{"verbose": "true"}}
	require.Equal(t, "true", a.Get("verbose"))
}

func TestGet_MissingFlag(t *testing.T) {
	a := Args{Flags: map[string]string{"verbose": "true"}}
	require.Equal(t, "", a.Get("missing"))
}

func TestHas_Present(t *testing.T) {
	a := Args{Flags: map[string]string{"verbose": "true"}}
	require.True(t, a.Has("verbose"))
}

func TestHas_Absent(t *testing.T) {
	a := Args{Flags: map[string]string{"verbose": "true"}}
	require.False(t, a.Has("missing"))
}

func TestString(t *testing.T) {
	a := Args{
		Positionals: []string{"node-1"},
		Flags:       map[string]string{"verbose": "true"},
	}
	s := a.String()
	require.Contains(t, s, "node-1")
	require.Contains(t, s, "verbose")
}

func TestArgs_NilFlags(t *testing.T) {
	a := Args{Flags: nil}
	require.Equal(t, "", a.Get("anything"))
	require.False(t, a.Has("anything"))
}
