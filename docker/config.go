package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
)

// ConfigWithDecodedData is a helper struct with the decoded data included.
type ConfigWithDecodedData struct {
	Config swarm.Config
	Data   []byte
}

// ListConfigs retrieves all Docker Swarm configs.
func ListConfigs(ctx context.Context) ([]swarm.Config, error) {
	l().Debug("[ListConfigs] Listing all configs")

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	configs, err := cli.ConfigList(ctx, types.ConfigListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	l().Infof("[ListConfigs] Found %d configs", len(configs))
	return configs, nil
}

// InspectConfig fetches and returns the config data.
func InspectConfig(ctx context.Context, nameOrID string) (*ConfigWithDecodedData, error) {
	l().Debugf("[InspectConfig] Inspecting config: %s", nameOrID)

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	cfg, _, err := cli.ConfigInspectWithRaw(ctx, nameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect config %q: %w", nameOrID, err)
	}

	l().Infof("[InspectConfig] Config %q inspected successfully (size=%d bytes)", cfg.Spec.Name, len(cfg.Spec.Data))
	return &ConfigWithDecodedData{Config: cfg, Data: cfg.Spec.Data}, nil
}

// CreateConfigVersion creates a new config, optionally using labels to mark lineage.
func CreateConfigVersion(ctx context.Context, baseConfig swarm.Config, newData []byte) (swarm.Config, error) {
	newName := nextConfigVersionName(baseConfig.Spec.Name)
	l().Infof("[CreateConfigVersion] Creating new config version from %q → %q (size=%d bytes)",
		baseConfig.Spec.Name, newName, len(newData))

	cli, err := GetClient()
	if err != nil {
		return swarm.Config{}, err
	}
	defer cli.Close()

	spec := swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name: newName,
			Labels: map[string]string{
				"swarmcli.origin":  baseConfig.Spec.Name,
				"swarmcli.created": time.Now().UTC().Format(time.RFC3339),
			},
		},
		Data: newData,
	}

	id, err := cli.ConfigCreate(ctx, spec)
	if err != nil {
		l().Errorf("[CreateConfigVersion] Failed to create config %q: %v", newName, err)
		return swarm.Config{}, fmt.Errorf("failed to create config %q: %w", newName, err)
	}

	newCfg, _, err := cli.ConfigInspectWithRaw(ctx, id.ID)
	if err != nil {
		l().Errorf("[CreateConfigVersion] Created config %q but failed to re-inspect: %v", newName, err)
		return swarm.Config{}, fmt.Errorf("failed to inspect new config %q: %w", newName, err)
	}

	l().Infof("[CreateConfigVersion] Successfully created new config %q (ID=%s)", newCfg.Spec.Name, newCfg.ID)
	return newCfg, nil
}

// RotateConfigInServices replaces the old config with the new one across all services using it.
func RotateConfigInServices(ctx context.Context, oldCfg, newCfg swarm.Config) error {
	l().Infof("[RotateConfig] Starting rotation: %q → %q", oldCfg.Spec.Name, newCfg.Spec.Name)

	cli, err := GetClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	services, err := cli.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list services: %w", err)
	}

	affected := 0
	for _, svc := range services {
		needsUpdate := false
		spec := svc.Spec

		// Check configs
		for i, ref := range spec.TaskTemplate.ContainerSpec.Configs {
			if ref.ConfigName == oldCfg.Spec.Name {
				l().Debugf("[RotateConfig] Service %s uses %q → replacing with %q",
					svc.Spec.Name, oldCfg.Spec.Name, newCfg.Spec.Name)

				spec.TaskTemplate.ContainerSpec.Configs[i] = &swarm.ConfigReference{
					ConfigName: newCfg.Spec.Name,
					ConfigID:   newCfg.ID,
					File:       ref.File,
				}
				needsUpdate = true
			}
		}

		if needsUpdate {
			affected++
			l().Infof("[RotateConfig] Updating service %s", svc.Spec.Name)

			response, err := cli.ServiceUpdate(ctx, svc.ID, svc.Version, spec, types.ServiceUpdateOptions{})
			if err != nil {
				l().Errorf("[RotateConfig] Failed to update service %s: %v", svc.Spec.Name, err)
				return fmt.Errorf("failed to update service %q: %w", svc.Spec.Name, err)
			}

			if len(response.Warnings) > 0 {
				for _, w := range response.Warnings {
					l().Warnf("[RotateConfig] Warning for service %s: %s", svc.Spec.Name, w)
				}
			}
		}
	}

	if affected == 0 {
		l().Infof("[RotateConfig] No services were using config %q", oldCfg.Spec.Name)
	} else {
		l().Infof("[RotateConfig] Rotation complete — updated %d service(s)", affected)
	}
	return nil
}

// --- Helpers ---

func nextConfigVersionName(baseName string) string {
	if idx := strings.LastIndex(baseName, "-v"); idx != -1 {
		var v int
		if _, err := fmt.Sscanf(baseName[idx+2:], "%d", &v); err == nil {
			return fmt.Sprintf("%s-v%d", baseName[:idx], v+1)
		}
	}
	return fmt.Sprintf("%s-v2", baseName)
}
