// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import (
	"testing"

	_ "github.com/Eldara-Tech/swarmcli/commands/command"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/charts"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/config"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/network"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/node"
	_ "github.com/Eldara-Tech/swarmcli/commands/command/docker/secret"
	"github.com/Eldara-Tech/swarmcli/registry"

	"github.com/stretchr/testify/require"
)

// TestAllCommands_HaveDetail guards that every registered OSS command
// ships a per-command help Detail paragraph. Passthrough stubs (the OSS
// bootstrap stub) are exempt: their help is intentionally never shown.
func TestAllCommands_HaveDetail(t *testing.T) {
	for _, c := range registry.PrimaryCommands() {
		// Unwrap: CommandWithAliases embeds the Command interface, so
		// its own method set does not include Spec().
		spec, ok := registry.SpecOf(c.Command)
		require.Truef(t, ok, "%s must declare a Spec()", c.Name())
		if spec.Passthrough {
			continue
		}
		require.NotEmptyf(t, spec.Detail, "%s spec must set Detail", c.Name())
	}
}

func TestParseArgs_NoArgs(t *testing.T) {
	result, err := parseArgs(nil, nil)
	require.NoError(t, err)
	require.Empty(t, result.Positionals)
	require.Empty(t, result.Flags)
}

func TestParseArgs_Positionals(t *testing.T) {
	result, err := parseArgs([]string{"node-1", "node-2"}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"node-1", "node-2"}, result.Positionals)
	require.Empty(t, result.Flags)
}

func TestParseArgs_FlagBoolean(t *testing.T) {
	result, err := parseArgs([]string{"--verbose"}, nil)
	require.NoError(t, err)
	require.Equal(t, "true", result.Flags["verbose"])
}

func TestParseArgs_FlagWithValue(t *testing.T) {
	result, err := parseArgs([]string{"--limit=10"}, nil)
	require.NoError(t, err)
	require.Equal(t, "10", result.Flags["limit"])
}

func TestParseArgs_Mixed(t *testing.T) {
	result, err := parseArgs([]string{"node-1", "--verbose", "--limit=10", "node-2"}, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"node-1", "node-2"}, result.Positionals)
	require.Equal(t, "true", result.Flags["verbose"])
	require.Equal(t, "10", result.Flags["limit"])
}

func TestParseArgs_ValueFlag_Spaced(t *testing.T) {
	vf := map[string]bool{"host": true}
	result, err := parseArgs([]string{"--host", "localhost", "--force"}, vf)
	require.NoError(t, err)
	require.Equal(t, "localhost", result.Flags["host"])
	require.Equal(t, "true", result.Flags["force"]) // not a value flag
	require.Empty(t, result.Positionals)            // "localhost" consumed, not positional
}

func TestParseArgs_ValueFlag_Equals_StillWorks(t *testing.T) {
	vf := map[string]bool{"host": true}
	result, err := parseArgs([]string{"--host=localhost"}, vf)
	require.NoError(t, err)
	require.Equal(t, "localhost", result.Flags["host"])
}

func TestParseArgs_ValueFlag_MissingValue(t *testing.T) {
	vf := map[string]bool{"host": true}
	_, err := parseArgs([]string{"--host"}, vf)
	require.Error(t, err)
	require.Contains(t, err.Error(), "flag --host requires a value")
}

func TestParseInput_EmptyString(t *testing.T) {
	_, _, err := ParseInput("")
	require.ErrorIs(t, err, ErrEmptyCommand)
}

func TestParseInput_UnknownCommand(t *testing.T) {
	_, _, err := ParseInput("nonexistent_xyz_command")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown command")
}

func TestParseInput_KnownCommand(t *testing.T) {
	// "help" is a registered command via the blank imports above
	cmd, a, err := ParseInput("help")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "help", cmd.Name())
	require.Empty(t, a.Positionals)
}

// These exercise ParseInput's flag/positional separation. They use the
// passthrough "bootstrap" command (OSS stub) so arbitrary flags are not
// rejected by strict validation — the separation mechanics are what is
// under test here; strict validation is covered separately below.
func TestParseInput_CommandWithArgs(t *testing.T) {
	cmd, a, err := ParseInput("bootstrap --verbose")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "bootstrap", cmd.Name())
	require.Equal(t, "true", a.Flags["verbose"])
}

func TestParseInput_CommandWithPositionalArgs(t *testing.T) {
	cmd, a, err := ParseInput("bootstrap mynode --verbose")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "bootstrap", cmd.Name())
	require.Equal(t, []string{"mynode"}, a.Positionals)
	require.Equal(t, "true", a.Flags["verbose"])
}

func TestParseInput_HelpFlag(t *testing.T) {
	// "node" is registered via the blank imports above and has a spec.
	cmd, _, err := ParseInput("node --help")
	var helpErr ErrHelpRequested
	require.ErrorAs(t, err, &helpErr)
	require.Equal(t, "node", helpErr.Cmd.Name())
	require.Nil(t, cmd)
}

func TestParseInput_HelpShortDash(t *testing.T) {
	_, _, err := ParseInput("node -h")
	var helpErr ErrHelpRequested
	require.ErrorAs(t, err, &helpErr)
	require.Equal(t, "node", helpErr.Cmd.Name())
}

func TestParseInput_HelpBeatsUnknownFlag(t *testing.T) {
	_, _, err := ParseInput("node --help --bogus")
	var helpErr ErrHelpRequested
	require.ErrorAs(t, err, &helpErr)
}

func TestParseInput_UnknownFlagRejected(t *testing.T) {
	_, _, err := ParseInput("node --bogus")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown flag --bogus for :node")
}

// :charts takes no flags, so its declared-but-flagless spec must reject every
// one of them. A command that declares no spec at all skips validateFlags
// instead, which is the failure this guards against.
func TestParseInput_ChartsIsStrict(t *testing.T) {
	cmd, _, err := ParseInput("charts")
	require.NoError(t, err)
	require.Equal(t, "charts", cmd.Name())

	_, _, err = ParseInput("charts --bogus")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown flag --bogus for :charts")
}

func TestParseInput_PassthroughSkipsValidation(t *testing.T) {
	// bootstrap (OSS stub) is Passthrough: --upgrade is not rejected and
	// --help is not intercepted; everything reaches Execute.
	cmd, a, err := ParseInput("bootstrap --upgrade")
	require.NoError(t, err)
	require.Equal(t, "bootstrap", cmd.Name())
	require.Equal(t, "true", a.Flags["upgrade"])

	_, _, err = ParseInput("bootstrap --help")
	require.NoError(t, err)
}
