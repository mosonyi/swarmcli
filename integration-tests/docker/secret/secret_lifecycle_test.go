//go:build integration

package secret

import (
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListSecrets(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	e := newTestEnv(t)
	defer e.cleanupAll(t)

	_, err := docker.ListSecrets(e.ctx)
	require.NoError(t, err, "ListSecrets should not error")
}

func TestCreateSecret(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	e := newTestEnv(t)
	defer e.cleanupAll(t)

	name := uniqueName("test_secret")
	sec := e.createSecret(t, name, "supersecretvalue")

	t.Logf("Created secret %s (ID=%s)", sec.Spec.Name, sec.ID)
	require.Equal(t, name, sec.Spec.Name)

	// Verify via inspect
	inspected, err := docker.InspectSecret(e.ctx, sec.Spec.Name)
	require.NoError(t, err)
	require.Equal(t, sec.ID, inspected.Secret.ID)
}

func TestCreateSecretVersion(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	e := newTestEnv(t)
	defer e.cleanupAll(t)

	name := uniqueName("test_secret_ver")
	base := e.createSecret(t, name, "v1data")

	newSec, err := docker.CreateSecretVersion(e.ctx, base, []byte("v2data"))
	require.NoError(t, err)
	e.registerSecretCleanup(newSec.ID)

	require.Contains(t, newSec.Spec.Name, "-v2", "versioned name should contain -v2")
	t.Logf("Created version %s from %s", newSec.Spec.Name, base.Spec.Name)
}

func TestDeleteSecret_Unused(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	e := newTestEnv(t)
	defer e.cleanupAll(t)

	name := uniqueName("test_secret_del")
	e.createSecret(t, name, "todelete")

	err := docker.DeleteSecret(e.ctx, name)
	require.NoError(t, err, "deleting unused secret should succeed")

	// Verify it's gone
	_, err = docker.InspectSecret(e.ctx, name)
	require.Error(t, err, "secret should no longer exist")
}

func TestSecretJSON(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	e := newTestEnv(t)
	defer e.cleanupAll(t)

	name := uniqueName("test_secret_json")
	e.createSecret(t, name, "jsontest")

	inspected, err := docker.InspectSecret(e.ctx, name)
	require.NoError(t, err)

	raw, err := inspected.JSON()
	require.NoError(t, err)
	require.Contains(t, string(raw), "write-only")

	pretty, err := inspected.PrettyJSON()
	require.NoError(t, err)
	require.Contains(t, string(pretty), "\n")
}
