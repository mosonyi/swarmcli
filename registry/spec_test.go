// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package registry

import (
	"swarmcli/args"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type specCommand struct {
	name string
	spec CommandSpec
}

func (s *specCommand) Name() string                       { return s.name }
func (s *specCommand) Description() string                { return "spec cmd" }
func (s *specCommand) Execute(_ any, _ args.Args) tea.Cmd { return nil }
func (s *specCommand) Spec() CommandSpec                  { return s.spec }

func TestSpecOf_NoSpec(t *testing.T) {
	_, ok := SpecOf(&mockCommand{name: "test_nospec"})
	require.False(t, ok)
}

func TestSpecOf_WithSpec(t *testing.T) {
	want := CommandSpec{Usage: "<x>", Flags: []FlagSpec{{Name: "f"}}}
	spec, ok := SpecOf(&specCommand{name: "test_withspec", spec: want})
	require.True(t, ok)
	require.Equal(t, want, spec)
}

func TestSpecOf_AliasDelegatesToPrimary(t *testing.T) {
	want := CommandSpec{Usage: "<primary>"}
	Register(&specCommand{name: "test_spec_primary", spec: want})
	Register(&mockAlias{name: "test_spec_alias", aliasOfCmd: "test_spec_primary"})
	defer cleanup("test_spec_primary", "test_spec_alias")

	alias, ok := Get("test_spec_alias")
	require.True(t, ok)
	spec, ok := SpecOf(alias)
	require.True(t, ok)
	require.Equal(t, want, spec)
}

func TestDistance(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"force", "force", 0},
		{"force", "frce", 1},
		{"host", "hosts", 1},
		{"upgrade", "upgrad", 1},
		{"db", "port", 4},
		{"", "abc", 3},
		{"abc", "", 3},
	}
	for _, c := range cases {
		require.Equal(t, c.want, Distance(c.a, c.b), "Distance(%q,%q)", c.a, c.b)
	}
}
