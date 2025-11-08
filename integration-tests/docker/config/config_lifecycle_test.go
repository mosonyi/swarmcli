//go:build integration

package config

import (
	"context"
	"testing"

	"swarmcli/docker"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestConfigLifecycle(t *testing.T) {
	ctx := context.Background()

	cli, err := docker.GetClient()
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	// --- 1. List configs ---
	t.Run("ListConfigs", func(t *testing.T) {
		_, err := docker.ListConfigs(ctx)
		require.NoError(t, err)
	})

	// --- 2. Create a config ---
	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: "demo_config-v1",
			Labels: map[string]string{
				"test": "integration",
			},
		},
		Data: []byte("hello world"),
	}
	cfgID, err := cli.ConfigCreate(ctx, spec)
	require.NoError(t, err, "ConfigCreate should succeed")

	t.Cleanup(func() { _ = cli.ConfigRemove(ctx, cfgID.ID) })

	cfg, _, err := cli.ConfigInspectWithRaw(ctx, cfgID.ID)
	require.NoError(t, err)
	require.Equal(t, "demo_config-v1", cfg.Spec.Name)

	// --- 3. CreateConfigVersion ---
	t.Run("CreateConfigVersion", func(t *testing.T) {
		newData := []byte("hello updated")
		newCfg, err := docker.CreateConfigVersion(ctx, cfg, newData)
		require.NoError(t, err)
		require.Contains(t, newCfg.Spec.Name, "-v")

		t.Cleanup(func() { _ = cli.ConfigRemove(ctx, newCfg.ID) })

		reloaded, _, err := cli.ConfigInspectWithRaw(ctx, newCfg.ID)
		require.NoError(t, err)
		require.Equal(t, newData, reloaded.Spec.Data)

		// Create another version to test unique naming
		nextCfg, err := docker.CreateConfigVersion(ctx, cfg, []byte("hello again"))
		require.NoError(t, err)
		require.NotEqual(t, newCfg.Spec.Name, nextCfg.Spec.Name)
		t.Cleanup(func() { _ = cli.ConfigRemove(ctx, nextCfg.ID) })
	})

	// --- 4. Service rotation ---
	t.Run("RotateConfigInServices", func(t *testing.T) {
		// Setup original service
		serviceSpec := swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: "demo_service",
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "traefik/whoami:v1.10",
					Configs: []*swarm.ConfigReference{{
						ConfigID:   cfg.ID,
						ConfigName: cfg.Spec.Name,
						File: &swarm.ConfigReferenceFileTarget{
							Name: "demo.conf",
							UID:  "0",
							GID:  "0",
							Mode: 0444,
						},
					}},
				},
			},
		}

		svcResp, err := cli.ServiceCreate(ctx, serviceSpec, types.ServiceCreateOptions{})
		require.NoError(t, err)
		t.Cleanup(func() { _ = cli.ServiceRemove(ctx, svcResp.ID) })

		newData := []byte("hello rotated")
		newCfg, err := docker.CreateConfigVersion(ctx, cfg, newData)
		require.NoError(t, err)
		t.Cleanup(func() { _ = cli.ConfigRemove(ctx, newCfg.ID) })

		err = docker.RotateConfigInServices(ctx, cfg, newCfg)
		require.NoError(t, err)

		svcAfter, _, err := cli.ServiceInspectWithRaw(ctx, svcResp.ID, types.ServiceInspectOptions{})
		require.NoError(t, err)

		foundNew := false
		for _, ref := range svcAfter.Spec.TaskTemplate.ContainerSpec.Configs {
			if ref.ConfigName == newCfg.Spec.Name {
				foundNew = true
			}
			require.NotEqual(t, cfg.Spec.Name, ref.ConfigName, "should not reference old config")
		}
		require.True(t, foundNew, "should reference new config")

		// Re-rotation should be a no-op
		err = docker.RotateConfigInServices(ctx, newCfg, newCfg)
		require.NoError(t, err, "rotation with identical configs should not fail")

		// Rotation with no services should be a no-op
		err = cli.ServiceRemove(ctx, svcResp.ID)
		require.NoError(t, err)
		err = docker.RotateConfigInServices(ctx, cfg, newCfg)
		require.NoError(t, err, "rotation with no active services should not fail")
	})
}
