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

	// --- 1. List configs ---
	_, err := docker.ListConfigs(ctx)
	require.NoError(t, err, "ListConfigs should not fail")

	// Create a new config for testing
	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: "demo_config-v1",
			Labels: map[string]string{
				"test": "integration",
			},
		},
		Data: []byte("hello world"),
	}

	cli, err := docker.GetClient()
	require.NoError(t, err)
	defer cli.Close()

	id, err := cli.ConfigCreate(ctx, spec)
	require.NoError(t, err, "ConfigCreate should succeed")

	cfg, _, err := cli.ConfigInspectWithRaw(ctx, id.ID)
	require.NoError(t, err, "ConfigInspect should succeed")
	require.Equal(t, "demo_config-v1", cfg.Spec.Name)

	// --- 2. CreateConfigVersion ---
	newData := []byte("hello updated")
	newCfg, err := docker.CreateConfigVersion(ctx, cfg, newData)
	require.NoError(t, err, "CreateConfigVersion should succeed")
	require.Contains(t, newCfg.Spec.Name, "-v", "new config name should have a version suffix")

	// --- 3. RotateConfigInServices ---
	// Create a temporary service using the original config
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
	require.NoError(t, err, "ServiceCreate should succeed")

	_, _, err = cli.ServiceInspectWithRaw(ctx, svcResp.ID, types.ServiceInspectOptions{})
	require.NoError(t, err, "ServiceInspect should succeed")

	err = docker.RotateConfigInServices(ctx, cfg, newCfg)
	require.NoError(t, err, "RotateConfigInServices should succeed")

	// --- Verify rotation ---
	svcAfter, _, err := cli.ServiceInspectWithRaw(ctx, svcResp.ID, types.ServiceInspectOptions{})
	require.NoError(t, err)

	found := false
	for _, ref := range svcAfter.Spec.TaskTemplate.ContainerSpec.Configs {
		if ref.ConfigName == newCfg.Spec.Name {
			found = true
			break
		}
	}
	require.True(t, found, "service should now reference new config")

	// --- Cleanup ---
	_ = cli.ServiceRemove(ctx, svcResp.ID)
	_ = cli.ConfigRemove(ctx, cfg.ID)
	_ = cli.ConfigRemove(ctx, newCfg.ID)
}
