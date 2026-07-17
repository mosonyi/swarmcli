// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// withEngineVersion sets the build-stamped engine version for one test and
// restores it afterwards.
func withEngineVersion(t *testing.T, v string) {
	t.Helper()
	prev := engineVersion
	engineVersion = v
	t.Cleanup(func() { engineVersion = prev })
}

func TestEngineVersion(t *testing.T) {
	withEngineVersion(t, "1.13.0")
	require.Equal(t, "1.13.0", EngineVersion())
}

func TestCheckCompat(t *testing.T) {
	tests := []struct {
		name       string
		engine     string
		constraint string
		want       CompatStatus
		reason     string // substring; empty means Reason must be empty too
	}{
		{name: "nothing declared", engine: "1.12.0", constraint: "", want: CompatUnknown},
		{name: "satisfied exactly", engine: "1.13.0", constraint: ">= 1.13.0", want: CompatOK},
		{name: "satisfied by newer", engine: "1.14.2", constraint: ">= 1.13.0", want: CompatOK},
		{name: "unsatisfied", engine: "1.12.0", constraint: ">= 1.13.0", want: CompatIncompatible},
		{name: "unsatisfied across a major", engine: "0.9.0", constraint: ">= 1.13.0", want: CompatIncompatible},
		{name: "caret range satisfied", engine: "1.13.4", constraint: "^1.13.0", want: CompatOK},
		{name: "caret range unsatisfied", engine: "2.0.0", constraint: "^1.13.0", want: CompatIncompatible},
		{name: "compound range satisfied", engine: "1.13.0", constraint: ">= 1.13.0, < 2.0.0", want: CompatOK},
		{name: "engine carries a v prefix", engine: "v1.13.0", constraint: ">= 1.13.0", want: CompatOK},

		// A release candidate carries its release's chart features. SemVer
		// constraints exclude prereleases by default, so without the core-version
		// strip these would report incompatible exactly while a release is being
		// tested.
		{name: "rc satisfies its own release", engine: "1.13.0-rc1", constraint: ">= 1.13.0", want: CompatOK},
		{name: "rc of a newer minor satisfies", engine: "v1.14.0-rc2", constraint: ">= 1.13.0", want: CompatOK},
		{name: "rc below the floor still fails", engine: "1.12.0-rc1", constraint: ">= 1.13.0", want: CompatIncompatible},
		{name: "build metadata is ignored", engine: "1.13.0+abc123", constraint: ">= 1.13.0", want: CompatOK},

		{
			name:   "unparseable constraint is ignored, not fatal",
			engine: "1.12.0", constraint: "newer than 1.13 please",
			want: CompatUnknown, reason: "not a valid SemVer constraint",
		},
		{
			name:   "unstamped build cannot check",
			engine: "", constraint: ">= 1.13.0",
			want: CompatUnknown, reason: "reports no chart-engine version",
		},
		{
			name:   "unparseable engine cannot check",
			engine: "not-a-version", constraint: ">= 1.13.0",
			want: CompatUnknown, reason: "is not valid SemVer",
		},
		{
			name:   "over-long constraint is refused before parsing",
			engine: "1.12.0", constraint: ">= 1.13.0" + strings.Repeat(" ", maxConstraintLen),
			want: CompatUnknown, reason: "longer than 256 bytes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			withEngineVersion(t, tc.engine)

			f := CheckCompat(Chartfile{Name: "demo", Version: "0.2.0", SwarmcliVersion: tc.constraint})

			require.Equal(t, tc.want, f.Status)
			if tc.reason == "" {
				require.Empty(t, f.Reason)
			} else {
				require.Contains(t, f.Reason, tc.reason)
			}
			// The finding always carries enough to render a message.
			require.Equal(t, "demo 0.2.0", f.Chart)
			require.Equal(t, tc.constraint, f.Required)
			require.Equal(t, tc.engine, f.Engine)
		})
	}
}

// CheckCompat is a compatibility aid, not a security boundary: a constraint it
// cannot use must never become a failure.
func TestCheckCompatNeverBlocksOnUnusableInput(t *testing.T) {
	withEngineVersion(t, "1.12.0")
	for _, c := range []string{"", "  ", ">>>", "1.2.3.4.5", "not a constraint", strings.Repeat(">", 5000)} {
		f := CheckCompat(Chartfile{Name: "demo", Version: "0.1.0", SwarmcliVersion: c})
		require.NotEqual(t, CompatIncompatible, f.Status, "constraint %q must not block", c)
	}
}

func TestCompatFindingMessage(t *testing.T) {
	f := CompatFinding{Chart: "traefik 0.2.0", Required: ">= 1.13.0", Engine: "1.12.0", Status: CompatIncompatible}
	const want = "chart traefik 0.2.0 requires swarmcli >= 1.13.0; this build provides 1.12.0"

	t.Run("same version names it once", func(t *testing.T) {
		require.Equal(t, want, f.Message("1.12.0"))
	})
	t.Run("a v-prefix skew is not a difference", func(t *testing.T) {
		require.Equal(t, want, f.Message("v1.12.0"))
	})
	t.Run("no binary version names only the engine", func(t *testing.T) {
		require.Equal(t, want, f.Message(""))
	})
	// The case the whole design exists for: a binary embedding an engine other
	// than its own version must name both, or it cites a release the reader
	// cannot map to anything they installed.
	t.Run("differing versions name both", func(t *testing.T) {
		require.Equal(t,
			"chart traefik 0.2.0 requires swarmcli >= 1.13.0; this build provides chart engine 1.12.0 (swarmcli 1.12.1)",
			f.Message("1.12.1"))
	})
}

func TestCompatFindingMessageUnstampedEngine(t *testing.T) {
	f := CompatFinding{Chart: "traefik 0.2.0", Required: ">= 1.13.0"}
	require.Equal(t,
		"chart traefik 0.2.0 requires swarmcli >= 1.13.0; this build reports no chart-engine version",
		f.Message("1.12.0"))
}

func TestCompatHint(t *testing.T) {
	incompatible := CompatFinding{
		Chart: "traefik 0.2.0", Required: ">= 1.13.0", Engine: "1.12.0", Status: CompatIncompatible,
	}
	hint := compatHint(incompatible)
	require.Contains(t, hint, "requires swarmcli >= 1.13.0")
	require.Contains(t, hint, "likely a consequence")

	// Anything else must add nothing to an unrelated error.
	for _, f := range []CompatFinding{{Status: CompatOK}, {Status: CompatUnknown}} {
		require.Empty(t, compatHint(f))
	}
}
