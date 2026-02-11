//go:build integration

package secret

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"testing"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

type testEnv struct {
	ctx     context.Context
	cli     *client.Client
	cleanup []func()
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()
	cli, err := docker.GetClient()
	require.NoError(t, err)
	return &testEnv{ctx: ctx, cli: cli}
}

func (e *testEnv) cleanupAll(t *testing.T) {
	for _, fn := range e.cleanup {
		fn()
	}
	docker.ResetClient()
}

func (e *testEnv) registerSecretCleanup(id string) {
	e.cleanup = append(e.cleanup, func() {
		_ = e.cli.SecretRemove(e.ctx, id)
	})
}

func (e *testEnv) createSecret(t *testing.T, name, data string) swarm.Secret {
	t.Helper()

	spec := swarm.SecretSpec{
		Annotations: swarm.Annotations{Name: name},
		Data:        []byte(data),
	}

	resp, err := e.cli.SecretCreate(e.ctx, spec)
	require.NoError(t, err, "failed to create secret %s", name)

	e.registerSecretCleanup(resp.ID)

	sec, _, err := e.cli.SecretInspectWithRaw(e.ctx, resp.ID)
	require.NoError(t, err, "failed to inspect secret %s", name)

	return sec
}

func uniqueName(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}
