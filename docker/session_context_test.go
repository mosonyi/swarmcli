// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// stubActiveContext replaces the active-context seam with a value the test
// controls, and clears both the pin and DOCKER_CONTEXT so the case starts from
// a known state. The returned function moves what the config file would say.
func stubActiveContext(t *testing.T, initial string) func(string) {
	t.Helper()
	orig := activeContextFn
	current := initial
	activeContextFn = func() (string, error) { return current, nil }
	ResetSessionContext()
	t.Cleanup(func() {
		activeContextFn = orig
		ResetSessionContext()
	})
	// Inherited from the developer's shell, this would outrank the stub.
	t.Setenv(envContextVar, "")
	return func(next string) { current = next }
}

// TestSessionContext_DoesNotFollowTheConfigFile is the regression test for
// #611. swarmcli used to answer "which context?" per call, so a
// `docker context use` in another terminal moved the exec-based half of the
// app — logs, deploy, stack rm — while the SDK client stayed connected to the
// context the session started in, and the two addressed different swarms.
func TestSessionContext_DoesNotFollowTheConfigFile(t *testing.T) {
	move := stubActiveContext(t, "swarm-a")

	pinned, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", pinned)

	move("swarm-b")

	got, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", got, "the pin must not follow the config file")
}

// TestResolvers_AllReportThePin covers the other half of the same defect: three
// exported resolvers used to make the same lookup independently, so it was
// possible for them to disagree with each other as well as with the client.
func TestResolvers_AllReportThePin(t *testing.T) {
	move := stubActiveContext(t, "swarm-a")
	_, err := SessionContext()
	require.NoError(t, err)

	move("swarm-b")

	fromEnv, err := GetContextFromEnv()
	require.NoError(t, err)
	current, err := GetCurrentContext()
	require.NoError(t, err)
	forStacks, err := GetDockerContext()
	require.NoError(t, err)

	require.Equal(t, "swarm-a", fromEnv)
	require.Equal(t, "swarm-a", current)
	require.Equal(t, "swarm-a", forStacks)
}

// TestSessionContext_EnvironmentWinsAtResolution keeps DOCKER_CONTEXT's meaning:
// it selects the context at startup. Setting it later does not move a session
// that is already running, which is the same promise made to a config-file
// switch.
func TestSessionContext_EnvironmentWinsAtResolution(t *testing.T) {
	stubActiveContext(t, "from-config")
	t.Setenv(envContextVar, "from-env")

	pinned, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "from-env", pinned)

	t.Setenv(envContextVar, "changed-later")

	got, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "from-env", got)
}

// TestSetSessionContext_MovesThePin — a switch made inside swarmcli is the one
// thing that moves it.
func TestSetSessionContext_MovesThePin(t *testing.T) {
	stubActiveContext(t, "swarm-a")
	_, err := SessionContext()
	require.NoError(t, err)

	SetSessionContext("swarm-b")

	got, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-b", got)
}

// TestSetSessionContext_IgnoresAnEmptyName — an empty name is a failed lookup
// somewhere upstream, and adopting it would leave every shell-out without a
// --context and back on the config file.
func TestSetSessionContext_IgnoresAnEmptyName(t *testing.T) {
	stubActiveContext(t, "swarm-a")
	_, err := SessionContext()
	require.NoError(t, err)

	SetSessionContext("   ")

	got, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", got)
}

// TestConfigFileContext_ReadsPastThePin — the drift check needs the live
// answer, and it is the only caller that does.
func TestConfigFileContext_ReadsPastThePin(t *testing.T) {
	move := stubActiveContext(t, "swarm-a")
	_, err := SessionContext()
	require.NoError(t, err)

	move("swarm-b")

	live, err := ConfigFileContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-b", live)

	pinned, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", pinned, "reading the live value must not move the pin")
}

// TestSessionContext_AFailedLookupIsNotCached — pinning a failure would make
// the session permanently unable to name a context, with no way back short of
// a restart.
func TestSessionContext_AFailedLookupIsNotCached(t *testing.T) {
	orig := activeContextFn
	ResetSessionContext()
	t.Cleanup(func() { activeContextFn = orig; ResetSessionContext() })
	t.Setenv(envContextVar, "")

	failing := true
	activeContextFn = func() (string, error) {
		if failing {
			return "", errors.New("docker not running")
		}
		return "swarm-a", nil
	}

	_, err := SessionContext()
	require.Error(t, err)

	failing = false

	got, err := SessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", got)
}

// TestEnvPinsContext — the drift check and ValidateContext both key off this,
// and disagreeing would either prompt for a switch the switcher then refuses,
// or refuse one nothing warned about.
func TestEnvPinsContext(t *testing.T) {
	t.Setenv(envContextVar, "")
	require.False(t, EnvPinsContext())

	t.Setenv(envContextVar, "swarm-a")
	require.True(t, EnvPinsContext())

	t.Setenv(envContextVar, "   ")
	require.False(t, EnvPinsContext(), "whitespace names no context")
}

// TestInitSessionContext_IsIdempotent — both entry points may call it, and a
// second call must report the pin rather than resolve a second time and
// possibly land somewhere else.
func TestInitSessionContext_IsIdempotent(t *testing.T) {
	move := stubActiveContext(t, "swarm-a")

	first, err := InitSessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", first)

	move("swarm-b")

	second, err := InitSessionContext()
	require.NoError(t, err)
	require.Equal(t, "swarm-a", second)
}

// TestActiveContextName_NeedsNoDockerCLI is the regression guard for the fork.
//
// Resolving the active context used to run `docker context show`, which the
// drift check calls on every app tick — a process start every five seconds to
// read one field. PATH is emptied here, so a implementation that forks cannot
// pass: the answer has to come from the config file directly.
func TestActiveContextName_NeedsNoDockerCLI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envConfigDirVar, dir)
	t.Setenv(envHostVar, "")
	t.Setenv("PATH", "")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"currentContext":"swarm-b","auths":{}}`), 0o600))

	got, err := activeContextName()
	require.NoError(t, err)
	require.Equal(t, "swarm-b", got)
}

func TestActiveContextName_MatchesDockersOwnAnswer(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  string // "" writes no file at all
		dirTrap bool   // put a directory where config.json should be, so the read fails
		host    string
		want    string
		wantErr bool
	}{
		{name: "a context the operator selected", config: `{"currentContext":"swarm-b"}`, want: "swarm-b"},
		{name: "no config file yet", want: defaultContextName},
		{name: "a config file that names none", config: `{"auths":{}}`, want: defaultContextName},
		// `docker context show` prints the whitespace verbatim and the old
		// fork trimmed it to "", which named nothing resolvable. Naming
		// nothing is what `default` means, so it is answered directly.
		{name: "a name that is only whitespace names nothing", config: `{"currentContext":"  "}`, want: defaultContextName},
		{
			// docker resolves DOCKER_HOST ahead of everything and pins `default`
			// to it. swarmcli does not honour the variable, but it has to agree
			// about the name or the drift check reports a switch nobody made.
			name: "DOCKER_HOST outranks the config file", config: `{"currentContext":"swarm-b"}`,
			host: "tcp://10.0.0.1:2376", want: defaultContextName,
		},
		{
			// `docker context show` warns and exits 0 with "default" here, so
			// failing the session would make swarmcli more broken than the CLI
			// for a file the CLI tolerates.
			name: "a malformed config file degrades as docker does", config: `{"currentContext":`, want: defaultContextName,
		},
		{
			// A file that exists and cannot be read is where Docker stops too,
			// and it is the one case worth surfacing: answering `default` would
			// invent an agreement the caller has no way to check.
			name: "a config file that cannot be read is an error", dirTrap: true, wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv(envConfigDirVar, dir)
			t.Setenv(envHostVar, tc.host)
			switch {
			case tc.dirTrap:
				require.NoError(t, os.Mkdir(filepath.Join(dir, "config.json"), 0o700))
			case tc.config != "":
				require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"), []byte(tc.config), 0o600))
			}

			got, err := activeContextName()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

// DOCKER_CONTEXT is still resolved ahead of the config file, as it was when the
// lookup forked.
func TestResolveContextName_TheEnvironmentStillOutranksTheFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envConfigDirVar, dir)
	t.Setenv(envHostVar, "")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.json"),
		[]byte(`{"currentContext":"from-file"}`), 0o600))
	t.Setenv(envContextVar, "from-env")

	got, err := resolveContextName()
	require.NoError(t, err)
	require.Equal(t, "from-env", got)
}
