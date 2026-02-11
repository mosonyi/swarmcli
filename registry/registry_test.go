package registry

import (
	"sort"
	"swarmcli/args"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

type mockCommand struct {
	name string
	desc string
}

func (m *mockCommand) Name() string                                { return m.name }
func (m *mockCommand) Description() string                         { return m.desc }
func (m *mockCommand) Execute(_ any, _ args.Args) tea.Cmd { return nil }

func cleanup(names ...string) {
	for _, n := range names {
		delete(apiRegistry, n)
	}
}

func TestRegister_And_Get(t *testing.T) {
	cmd := &mockCommand{name: "test_reg_get", desc: "test"}
	Register(cmd)
	defer cleanup("test_reg_get")

	got, ok := Get("test_reg_get")
	require.True(t, ok)
	require.Equal(t, "test_reg_get", got.Name())
}

func TestGet_NotFound(t *testing.T) {
	_, ok := Get("nonexistent_command_xyz")
	require.False(t, ok)
}

func TestAll_ReturnsRegistered(t *testing.T) {
	cmd := &mockCommand{name: "test_all_cmd", desc: "test"}
	Register(cmd)
	defer cleanup("test_all_cmd")

	all := All()
	found := false
	for _, c := range all {
		if c.Name() == "test_all_cmd" {
			found = true
			break
		}
	}
	require.True(t, found, "All() should contain registered command")
}

func TestSuggest_MatchingPrefix(t *testing.T) {
	Register(&mockCommand{name: "test_suggest_alpha"})
	Register(&mockCommand{name: "test_suggest_beta"})
	defer cleanup("test_suggest_alpha", "test_suggest_beta")

	suggestions := Suggest("test_suggest_a")
	require.Contains(t, suggestions, "test_suggest_alpha")
	require.NotContains(t, suggestions, "test_suggest_beta")
}

func TestSuggest_EmptyPrefix(t *testing.T) {
	Register(&mockCommand{name: "test_suggest_empty"})
	defer cleanup("test_suggest_empty")

	suggestions := Suggest("")
	require.Contains(t, suggestions, "test_suggest_empty")
}

func TestSuggest_NoMatch(t *testing.T) {
	suggestions := Suggest("zzz_no_match_prefix")
	sort.Strings(suggestions) // deterministic
	require.Empty(t, suggestions)
}

func TestRegister_OverwritesSameName(t *testing.T) {
	Register(&mockCommand{name: "test_overwrite", desc: "first"})
	Register(&mockCommand{name: "test_overwrite", desc: "second"})
	defer cleanup("test_overwrite")

	cmd, ok := Get("test_overwrite")
	require.True(t, ok)
	require.Equal(t, "second", cmd.Description())
}
