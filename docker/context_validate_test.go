// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// validateSeams records the order of the side effects ValidateContext performs,
// which is the whole of what it gets wrong when the client is not reset: the
// probe has to happen *after* the switch and *after* the client is dropped, or
// it describes the context being left rather than the one being entered.
type validateSeams struct {
	trace    []string
	current  string
	curErr   error
	useErr   error
	probeErr error
}

func stubValidateSeams(t *testing.T, s *validateSeams) {
	t.Helper()
	origUse, origReset := useContextFn, resetClientFn
	origCurrent, origProbe := currentContextFn, probeContextFn
	origSet := setContextFn
	t.Cleanup(func() {
		useContextFn, resetClientFn = origUse, origReset
		currentContextFn, probeContextFn = origCurrent, origProbe
		setContextFn = origSet
	})

	if s.current == "" {
		s.current = "old"
	}
	useContextFn = func(name string) error {
		s.trace = append(s.trace, "use:"+name)
		return s.useErr
	}
	resetClientFn = func() { s.trace = append(s.trace, "reset") }
	setContextFn = func(name string) { s.trace = append(s.trace, "pin:"+name) }
	currentContextFn = func() (string, error) { return s.current, s.curErr }
	probeContextFn = func(context.Context, string) error {
		s.trace = append(s.trace, "probe")
		return s.probeErr
	}
	// Inherited from the developer's shell, this would short-circuit the
	// function under test.
	t.Setenv("DOCKER_CONTEXT", "")
}

// Two pieces of process-wide state describe the previous context: the session
// pin every caller resolves through, and the cached client built from it.
// Probing before both have moved validates the context being left, which is how
// an unreachable target used to pass.
func TestValidateContext_MovesThePinAndDropsTheClientBeforeProbing(t *testing.T) {
	s := &validateSeams{}
	stubValidateSeams(t, s)

	require.NoError(t, ValidateContext(context.Background(), "new"))
	require.Equal(t, []string{"use:new", "pin:new", "reset", "probe"}, s.trace)
}

// A target that does not answer must leave the machine exactly as it was —
// original context current, and no client left over from the rejected one.
func TestValidateContext_UnreachableTargetRevertsAndDropsItsClient(t *testing.T) {
	boom := errors.New("failed to ping context new: connection refused")
	s := &validateSeams{probeErr: boom}
	stubValidateSeams(t, s)

	err := ValidateContext(context.Background(), "new")
	require.ErrorIs(t, err, boom, "the probe's reason must reach the caller unchanged")
	require.Equal(t,
		[]string{"use:new", "pin:new", "reset", "probe", "use:old", "pin:old", "reset"},
		s.trace)
}

// DOCKER_CONTEXT is resolved ahead of ~/.docker/config.json, so switching
// elsewhere would not change what this process talks to. Reporting success
// there is the lie; refuse before touching anything.
func TestValidateContext_RefusesWhenPinnedElsewhereByEnv(t *testing.T) {
	s := &validateSeams{}
	stubValidateSeams(t, s)
	t.Setenv("DOCKER_CONTEXT", "pinned")

	err := ValidateContext(context.Background(), "new")
	require.ErrorIs(t, err, ErrContextPinnedByEnv)
	require.Contains(t, err.Error(), "pinned")
	require.Contains(t, err.Error(), "new")
	require.Empty(t, s.trace, "nothing may be switched or reset when the switch cannot take effect")
}

// Pinning the very context being validated is not a conflict: the process is
// already there, and the caller only wants to know it works.
func TestValidateContext_EnvNamingTheSameContextIsNotAConflict(t *testing.T) {
	s := &validateSeams{}
	stubValidateSeams(t, s)
	t.Setenv("DOCKER_CONTEXT", "new")

	require.NoError(t, ValidateContext(context.Background(), "new"))
	require.Equal(t, []string{"use:new", "pin:new", "reset", "probe"}, s.trace)
}

// Without a context to revert to there is nothing safe to do, so the switch
// must not be attempted at all.
func TestValidateContext_UnknownCurrentContextSwitchesNothing(t *testing.T) {
	s := &validateSeams{curErr: errors.New("docker context show: exit 1")}
	stubValidateSeams(t, s)

	require.Error(t, ValidateContext(context.Background(), "new"))
	require.Empty(t, s.trace)
}

// A failed `docker context use` has moved nothing, so there is nothing to
// revert and no client to drop.
func TestValidateContext_FailedSwitchDoesNotRevert(t *testing.T) {
	s := &validateSeams{useErr: errors.New("no such context")}
	stubValidateSeams(t, s)

	require.Error(t, ValidateContext(context.Background(), "new"))
	require.Equal(t, []string{"use:new"}, s.trace)
}
