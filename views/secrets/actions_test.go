package secretsview

import (
	"context"
	"fmt"
	"testing"

	"swarmcli/docker"
	"swarmcli/views/view"

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
	mock.listServicesUsingSecretIDFn = func(_ context.Context, secretID string) ([]swarm.Service, error) {
		if secretID == "id-used" {
			return []swarm.Service{{ID: "svc1"}}, nil
		}
		return nil, nil
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
}

func TestGetUsedByStacksCmd_ReturnsUsedByMsg(t *testing.T) {
	mock := noopSecretOps()
	mock.inspectSecretFn = func(_ context.Context, _ string) (*docker.SecretWithDecodedData, error) {
		return &docker.SecretWithDecodedData{
			Secret: swarm.Secret{ID: "sec-id", Spec: swarm.SecretSpec{Annotations: swarm.Annotations{Name: "s1"}}},
		}, nil
	}
	mock.listServicesUsingSecretNameFn = func(_ context.Context, _ string) ([]swarm.Service, error) {
		return []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "web", Labels: map[string]string{"com.docker.stack.namespace": "mystack"}}}},
		}, nil
	}
	mock.listServicesUsingSecretIDFn = func(_ context.Context, _ string) ([]swarm.Service, error) {
		return nil, nil
	}

	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	cmd := m.getUsedByStacksCmd("s1")
	msg := runCmd(cmd)
	used, ok := msg.(usedByMsg)
	require.True(t, ok)
	require.Equal(t, "s1", used.SecretName)
	require.Len(t, used.UsedBy, 1)
	require.Equal(t, "mystack", used.UsedBy[0].StackName)
}

func TestCheckSecretsCmd_NoChange_ReturnsTick(t *testing.T) {
	mock := noopSecretOps()
	mock.listSecretsFn = func(_ context.Context) ([]swarm.Secret, error) {
		return nil, nil // empty = same as initial hash 0
	}
	m := testModel(func(m *Model) { m.deps.Secrets = mock })
	// lastSnapshot=0 for empty, ListSecrets returns empty too → hashes match
	cmd := m.checkSecretsCmd(0)
	msg := runCmd(cmd)
	// Should be a tick cmd (not secretsLoadedMsg)
	_, isLoaded := msg.(secretsLoadedMsg)
	require.False(t, isLoaded, "should not return secretsLoadedMsg when no change")
}
