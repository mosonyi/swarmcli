package config

import (
	"strings"
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestDeleteConfig(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	e := newTestEnv(t)
	defer e.cleanupAll(t)

	t.Run("Delete unused config succeeds", func(t *testing.T) {
		cfg := e.createConfig(t, uniqueName("delete_me"), "temporary data")
		err := docker.DeleteConfig(e.ctx, cfg.Spec.Name)
		require.NoError(t, err, "DeleteConfig should succeed for unused config")

		// Verify it’s actually gone
		_, err = docker.InspectConfig(e.ctx, cfg.Spec.Name)
		require.Error(t, err, "InspectConfig should fail for deleted config")
	})

	t.Run("Delete nonexistent config fails", func(t *testing.T) {
		err := docker.DeleteConfig(e.ctx, "nonexistent-config-xyz")
		require.Error(t, err, "DeleteConfig should fail for nonexistent config")
		require.True(t,
			strings.Contains(err.Error(), "failed to inspect config") &&
				strings.Contains(err.Error(), "not found"),
			"Expected wrapped inspect error mentioning not found, got: %v", err,
		)
	})

	t.Run("Delete config in use fails", func(t *testing.T) {
		// Create config and a service that uses it
		cfg := e.createConfig(t, uniqueName("inuse_config-v1"), "in-use")
		e.registerConfigCleanup(cfg.ID)

		serviceSpec := swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: uniqueName("demo_service"),
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
		svcResp, err := e.cli.ServiceCreate(e.ctx, serviceSpec, types.ServiceCreateOptions{})
		require.NoError(t, err)
		e.registerServiceCleanup(svcResp.ID)

		// Attempt delete should fail
		err = docker.DeleteConfig(e.ctx, cfg.Spec.Name)
		require.Error(t, err, "DeleteConfig should fail for in-use config")
		require.Contains(t, err.Error(), "still used by services", "Error should mention service usage")
	})

	t.Run("Delete config by ID also works", func(t *testing.T) {
		cfg := e.createConfig(t, uniqueName("delete_by_id"), "data")
		err := docker.DeleteConfig(e.ctx, cfg.ID)
		require.NoError(t, err, "DeleteConfig should work when deleting by ID")
	})
}
