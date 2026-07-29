// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package config

import (
	"github.com/Eldara-Tech/swarmcli/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"
	"testing"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestConfigLifecycle(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	e := newTestEnv(t)
	defer e.cleanupAll(t)

	t.Run("ListConfigs", func(t *testing.T) {
		_, err := docker.ListConfigs(e.ctx)
		require.NoError(t, err, "ListConfigs should not fail")
	})

	// ListConfigs does not inspect each config, because a real daemon returns
	// every field in the list response already. That is a claim about the
	// daemon, so only a real one can hold it: if it ever stopped being true the
	// configs view would lose its CREATED AT column and the chart engine would
	// decode empty release payloads (issue #510).
	t.Run("ListConfigsCarriesMetadataAndPayload", func(t *testing.T) {
		name := uniqueName("demo_config-listed")
		created := e.createConfig(t, name, "listed payload")

		cfgs, err := docker.ListConfigs(e.ctx)
		require.NoError(t, err)

		var found *swarm.Config
		for i, c := range cfgs {
			if c.ID == created.ID {
				found = &cfgs[i]
				break
			}
		}
		require.NotNil(t, found, "the config just created should be listed")
		require.False(t, found.CreatedAt.IsZero(), "the list response should carry CreatedAt")
		require.NotZero(t, found.Version.Index, "the list response should carry Version")
		require.Equal(t, []byte("listed payload"), found.Spec.Data, "the list response should carry the payload")
	})

	t.Run("CreateConfigVersion", func(t *testing.T) {
		orig := e.createConfig(t, uniqueName("demo_config-v1"), "hello world")

		newData := []byte("hello updated")
		newCfg, err := docker.CreateConfigVersion(e.ctx, orig, newData)
		require.NoError(t, err, "CreateConfigVersion should succeed")
		require.Contains(t, newCfg.Spec.Name, "-v", "new config name should have version suffix")

		e.registerConfigCleanup(newCfg.ID)
	})

	t.Run("RotateConfigInServices", func(t *testing.T) {
		orig := e.createConfig(t, uniqueName("demo_config-v1"), "initial")
		newData := []byte("updated data")
		newCfg, err := docker.CreateConfigVersion(e.ctx, orig, newData)
		require.NoError(t, err, "CreateConfigVersion should succeed")
		e.registerConfigCleanup(newCfg.ID)

		serviceSpec := swarm.ServiceSpec{
			Annotations: swarm.Annotations{
				Name: uniqueName("demo_service"),
			},
			TaskTemplate: swarm.TaskSpec{
				ContainerSpec: &swarm.ContainerSpec{
					Image: "traefik/whoami:v1.10",
					Configs: []*swarm.ConfigReference{{
						ConfigID:   orig.ID,
						ConfigName: orig.Spec.Name,
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

		svcResp, err := e.cli.ServiceCreate(e.ctx, serviceSpec, swarm.ServiceCreateOptions{})
		require.NoError(t, err, "ServiceCreate should succeed")
		e.registerServiceCleanup(svcResp.ID)

		err = docker.RotateConfigInServices(e.ctx, &orig, newCfg)
		require.NoError(t, err, "RotateConfigInServices should succeed")

		svcAfter, _, err := e.cli.ServiceInspectWithRaw(e.ctx, svcResp.ID, swarm.ServiceInspectOptions{})
		require.NoError(t, err)

		found := false
		for _, ref := range svcAfter.Spec.TaskTemplate.ContainerSpec.Configs {
			if ref.ConfigName == newCfg.Spec.Name {
				found = true
				break
			}
		}
		require.True(t, found, "service should now reference new config")
	})
}
