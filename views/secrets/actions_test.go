// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Eldara-Tech/swarmcli/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestLoadSecretsCmd_CallsListSecrets(t *testing.T) {
	called := false
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		called = true
		return []swarm.Secret{
			{ID: "id1", Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "s1"}}},
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.loadSecretsCmd()
	msg := runCmd(cmd)
	require.True(t, called)
	loaded, ok := msg.(secretsLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded, 1)
}

func TestLoadSecretsCmd_Error_ReturnsErrorMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return nil, fmt.Errorf("connection refused")
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.loadSecretsCmd()
	msg := runCmd(cmd)
	_, ok := msg.(errorMsg)
	require.True(t, ok)
}

func TestDeleteSecretCmd_CallsDeleteSecret(t *testing.T) {
	deleted := ""
	mock := noopSecretOps()
	mock.deleteSecretFn = func(_ context.Context, nameOrID string) error {
		deleted = nameOrID
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.deleteSecretCmd("my-secret")
	msg := runCmd(cmd)
	del, ok := msg.(secretDeletedMsg)
	require.True(t, ok)
	require.Equal(t, "my-secret", del.Name)
	require.Equal(t, "my-secret", deleted)
}

func TestDeleteSecretCmd_Error_ReturnsErrorMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.deleteSecretFn = func(_ context.Context, _ string) error {
		return fmt.Errorf("not found")
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.deleteSecretCmd("missing")
	msg := runCmd(cmd)
	_, ok := msg.(errorMsg)
	require.True(t, ok)
}

func TestInspectSecretCmd_NavigatesToInspect(t *testing.T) {
	mock := noopSecretOps()
	mock.inspectSecretFn = func(_ context.Context, nameOrID string) (*docker.SecretWithDecodedData, error) {
		return &docker.SecretWithDecodedData{
			Secret: swarm.Secret{
				ID:   "id1",
				Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: nameOrID}},
			},
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.inspectSecretCmd("s1")
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
	payload := nav.Payload.(map[string]any)
	require.Contains(t, payload["title"], "s1")
}

func TestComputeSecretUsedCmd_SetsUsedMap(t *testing.T) {
	mock := noopSecretOps()
	mock.servicesUsingSecretsFn = func(_ context.Context) (map[string][]swarm.Service, error) {
		return map[string][]swarm.Service{"id-used": {{ID: "svc1"}}}, nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })

	secs := []docker.SecretWithDecodedData{
		{Secret: swarm.Secret{ID: "id-used"}},
		{Secret: swarm.Secret{ID: "id-unused"}},
	}
	cmd := m.computeSecretUsedCmd(secs)
	msg := runCmd(cmd)
	usedMap, ok := msg.(usedStatusUpdatedMsg)
	require.True(t, ok)
	require.True(t, usedMap["id-used"])
	require.False(t, usedMap["id-unused"])
	// The point of the index: one lookup regardless of how many secrets the
	// swarm holds. Per-secret lookups each list every service.
	require.Equal(t, 1, mock.servicesUsingSecretsCalls)
}

// A secret the view has just created cannot be referenced yet, so building its
// row asks the daemon nothing at all.
func TestAddSecretDoesNotQueryServices(t *testing.T) {
	mock := noopSecretOps()
	m := testModel(func(m *Model) { m.deps.Secrets = mock })

	m.addSecret(docker.SecretWithDecodedData{
		Secret: swarm.Secret{ID: "fresh", Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "s-new"}}},
	})

	require.Zero(t, mock.servicesUsingSecretsCalls)
	require.NotEmpty(t, m.secretsList.Items)
	item := m.secretsList.Items[len(m.secretsList.Items)-1]
	require.Equal(t, "s-new", item.Name)
	require.False(t, item.Used)
	require.True(t, item.UsedKnown)
}

func TestGetUsedByStacksCmd_ReturnsUsedByMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.inspectSecretFn = func(_ context.Context, _ string) (*docker.SecretWithDecodedData, error) {
		return &docker.SecretWithDecodedData{
			Secret: swarm.Secret{ID: "sec-id", Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "s1"}}},
		}, nil
	}
	// One service reachable by name, another by ID: the index carries both keys,
	// which is what lets a single listing replace the two this used to merge.
	mock.servicesUsingSecretsFn = func(_ context.Context) (map[string][]swarm.Service, error) {
		return map[string][]swarm.Service{
			"s1": {
				{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "web", Labels: map[string]string{"com.docker.stack.namespace": "mystack"}}}},
			},
			"sec-id": {
				{ID: "svc2", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "api", Labels: map[string]string{"com.docker.stack.namespace": "mystack"}}}},
			},
		}, nil
	}

	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.getUsedByStacksCmd("s1")
	msg := runCmd(cmd)
	used, ok := msg.(usedByMsg)
	require.True(t, ok)
	require.Equal(t, "s1", used.SecretName)
	require.Len(t, used.UsedBy, 2)
	require.Equal(t, "mystack", used.UsedBy[0].StackName)
	require.Equal(t, "api", used.UsedBy[0].ServiceName, "sorted by stack then service")
	require.Equal(t, "web", used.UsedBy[1].ServiceName)
	require.Equal(t, 1, mock.servicesUsingSecretsCalls)
}

func TestCheckSecretsCmd_NoChange_ReturnsPollRetry(t *testing.T) {
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return nil, nil // empty = same as initial hash 0
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	// lastSnapshot=0 for empty, ListSecrets returns empty too → hashes match
	cmd := m.checkSecretsCmd(0)
	msg := runCmd(cmd)
	_, isLoaded := msg.(secretsLoadedMsg)
	require.False(t, isLoaded, "should not return secretsLoadedMsg when no change")
	_, isPollRetry := msg.(PollRetryMsg)
	require.True(t, isPollRetry, "should return PollRetryMsg when no change")
}

func TestCheckSecretsCmd_Timeout_ReturnsPollRetryMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return nil, context.DeadlineExceeded
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.checkSecretsCmd(0)
	msg := runCmd(cmd)
	_, ok := msg.(PollRetryMsg)
	require.True(t, ok)
}

func TestLoadSecretsCmd_Timeout_ReturnsErrorMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return nil, context.DeadlineExceeded
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.loadSecretsCmd()
	msg := runCmd(cmd)
	_, ok := msg.(errorMsg)
	require.True(t, ok)
}

func TestCheckSecretsCmd_InFlightGuard_SkipsDuplicate(t *testing.T) {
	called := false
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		called = true
		return nil, nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	m.polling.Store(true)
	cmd := m.checkSecretsCmd(0)
	msg := runCmd(cmd)
	_, ok := msg.(PollRetryMsg)
	require.True(t, ok)
	require.False(t, called, "listSecretsFn should not be called when polling is in flight")
}

func TestCheckSecretsCmd_HashMatchesAfterLoad(t *testing.T) {
	secrets := []swarm.Secret{
		{ID: "id1", Meta: swarm.Meta{Version: swarm.Version{Index: 1}}, Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "alpha"}}},
		{ID: "id2", Meta: swarm.Meta{Version: swarm.Version{Index: 2}}, Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "bravo"}}},
	}
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return secrets, nil
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })

	// Simulate initial load so m.lastSnapshot is set
	wrapped := make([]docker.SecretWithDecodedData, len(secrets))
	for i, s := range secrets {
		wrapped[i] = docker.SecretWithDecodedData{Secret: s}
	}
	m.Update(secretsLoadedMsg(wrapped))

	// Poll with the stored snapshot — same data must NOT trigger a reload
	cmd := m.checkSecretsCmd(m.lastSnapshot)
	msg := runCmd(cmd)
	_, isLoaded := msg.(secretsLoadedMsg)
	require.False(t, isLoaded, "hash from secretsLoadedMsg must match hash from checkSecretsCmd for identical data")
}

// captureLog bridges the package logger into a buffer at debug level, so a test
// can assert on everything the code under test writes. utils/log keeps the
// previous logger in package-private state, so cleanup installs a discarding
// bridge rather than restoring one; no other test here reads the log.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	swarmlog.InitSlog(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	t.Cleanup(func() {
		swarmlog.InitSlog(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))
	})
	return &buf
}

// A secret's own bytes must never reach the log at any level. The debug log is
// the file an operator attaches to a bug report, and lumberjack keeps five
// compressed rotations of it for fourteen days, so a line written here outlives
// the terminal the secret was typed into by a fortnight.
//
// Both creation paths are covered because each encodes its payload itself, and
// each carried its own copy of the line that printed it — fixing one proved
// nothing about the other. The unencoded cases are here to catch a log added to
// the branch that does no encoding, which is the obvious place for the next one
// to appear.
//
// The Contains assertion is not decoration. Without it, a bridge that captured
// nothing at all would satisfy every NotContains below and the test would pass
// while asserting nothing.
func TestCreateSecretNeverLogsThePayload(t *testing.T) {
	const payload = "hunter2-correct-horse-battery-staple"
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))

	file := filepath.Join(t.TempDir(), "secret.txt")
	require.NoError(t, os.WriteFile(file, []byte(payload), 0o600))

	for _, tc := range []struct {
		name string
		cmd  func(*Model) tea.Cmd
	}{
		{"from file", func(m *Model) tea.Cmd {
			return m.createSecretFromFileCmd("s-file", file, nil, true)
		}},
		{"from file, not encoded", func(m *Model) tea.Cmd {
			return m.createSecretFromFileCmd("s-file", file, nil, false)
		}},
		{"from content", func(m *Model) tea.Cmd {
			return m.createSecretFromContentCmd("s-content", []byte(payload), nil, true)
		}},
		{"from content, not encoded", func(m *Model) tea.Cmd {
			return m.createSecretFromContentCmd("s-content", []byte(payload), nil, false)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := captureLog(t)
			m := testModel(func(m *Model) { m.deps.Secrets = noopSecretOps() })

			require.IsType(t, secretCreatedMsg{}, runCmd(tc.cmd(m)))

			logged := buf.String()
			require.Contains(t, logged, "Creating secret",
				"the bridge captured none of this command's own lines, so the assertions below prove nothing")
			require.NotContains(t, logged, payload, "the secret's plaintext reached the log")
			require.NotContains(t, logged, encoded, "the secret's base64 reached the log")
		})
	}
}
