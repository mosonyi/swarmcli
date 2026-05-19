// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import (
	"testing"

	_ "swarmcli/commands/command"
	_ "swarmcli/commands/command/docker"
	_ "swarmcli/commands/command/docker/config"
	_ "swarmcli/commands/command/docker/network"
	_ "swarmcli/commands/command/docker/node"
	_ "swarmcli/commands/command/docker/secret"

	"github.com/stretchr/testify/require"
)

func TestParseArgs_NoArgs(t *testing.T) {
	result := parseArgs(nil)
	require.Empty(t, result.Positionals)
	require.Empty(t, result.Flags)
}

func TestParseArgs_Positionals(t *testing.T) {
	result := parseArgs([]string{"node-1", "node-2"})
	require.Equal(t, []string{"node-1", "node-2"}, result.Positionals)
	require.Empty(t, result.Flags)
}

func TestParseArgs_FlagBoolean(t *testing.T) {
	result := parseArgs([]string{"--verbose"})
	require.Equal(t, "true", result.Flags["verbose"])
}

func TestParseArgs_FlagWithValue(t *testing.T) {
	result := parseArgs([]string{"--limit=10"})
	require.Equal(t, "10", result.Flags["limit"])
}

func TestParseArgs_Mixed(t *testing.T) {
	result := parseArgs([]string{"node-1", "--verbose", "--limit=10", "node-2"})
	require.Equal(t, []string{"node-1", "node-2"}, result.Positionals)
	require.Equal(t, "true", result.Flags["verbose"])
	require.Equal(t, "10", result.Flags["limit"])
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
