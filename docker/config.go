package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// ConfigWithDecodedData is a helper struct with the decoded data included.
type ConfigWithDecodedData struct {
	Config swarm.Config
	Data   []byte
}

func (cfg *ConfigWithDecodedData) JSON() ([]byte, error) {
	type jsonConfig struct {
		Config     swarm.Config `json:"Config"`
		DataParsed any          `json:"DataParsed,omitempty"`
	}

	obj := jsonConfig{Config: cfg.Config}

	parsedMap := make(map[string]string)
	parsed := true

	for _, line := range strings.Split(string(cfg.Data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			parsed = false
			break
		}
		parsedMap[parts[0]] = parts[1]
	}

	if parsed && len(parsedMap) > 0 {
		// Sort keys for consistent ordering
		keys := make([]string, 0, len(parsedMap))
		for k := range parsedMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		ordered := make(map[string]string, len(parsedMap))
		for _, k := range keys {
			ordered[k] = parsedMap[k]
		}
		obj.DataParsed = ordered
	} else {
		obj.DataParsed = string(cfg.Data)
	}

	return json.Marshal(obj)
}

// PrettyJSON returns the JSON representation of the config,
// but pretty-printed (indented) for human-readable editing.
func (cfg *ConfigWithDecodedData) PrettyJSON() ([]byte, error) {
	raw, err := cfg.JSON()
	if err != nil {
		return nil, err
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return nil, err
	}

	return pretty.Bytes(), nil
}

// ListConfigs retrieves all Docker Swarm configs.
func ListConfigs(ctx context.Context) ([]swarm.Config, error) {
	l().Debug("[ListConfigs] Listing all configs")

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer closeCli(cli)

	configs, err := cli.ConfigList(ctx, swarm.ConfigListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}

	// 🔠 Sort configs alphabetically by name
	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Spec.Name < configs[j].Spec.Name
	})

	l().Infof("[ListConfigs] Found %d configs", len(configs))
	return configs, nil
}

func closeCli(cli *client.Client) {
	err := cli.Close()
	if err != nil {
		l().Errorf("failed to close client: %v", err)
	}
}

// InspectConfig fetches and returns the config data.
func InspectConfig(ctx context.Context, nameOrID string) (*ConfigWithDecodedData, error) {
	l().Debugf("[InspectConfig] Inspecting config: %s", nameOrID)

	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer closeCli(cli)

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
	defer closeCli(cli)

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

// RotateConfigInServices updates all services that reference oldCfg to use newCfg.
// If oldCfg is nil, it tries to infer affected services automatically based on labels or content.
func RotateConfigInServices(ctx context.Context, oldCfg *swarm.Config, newCfg swarm.Config) error {
	if newCfg.ID == "" {
		return fmt.Errorf("new config must have a valid ID")
	}

	client, err := GetClient()
	if err != nil {
		return fmt.Errorf("failed to get docker client: %w", err)
	}
	defer client.Close()

	// --- 1. Find affected services
	var services []swarm.Service
	if oldCfg != nil {
		services, err = listServicesUsingConfig(ctx, client, oldCfg.ID)
	} else {
		services, err = listServicesUsingConfigName(ctx, client, newCfg.Spec.Name)
	}
	if err != nil {
		return fmt.Errorf("failed to list services for rotation: %w", err)
	}

	if len(services) == 0 {
		l().Infof("No services found using config %s", newCfg.Spec.Name)
		return nil
	}

	// --- 2. Apply updates
	for _, svc := range services {
		updated := svc.Spec
		for i, cfgRef := range updated.TaskTemplate.ContainerSpec.Configs {
			// Match by old ID or by name if oldCfg is nil
			if (oldCfg != nil && cfgRef.ConfigID == oldCfg.ID) ||
				(oldCfg == nil && cfgRef.ConfigName == newCfg.Spec.Name) {
				updated.TaskTemplate.ContainerSpec.Configs[i].ConfigID = newCfg.ID
				updated.TaskTemplate.ContainerSpec.Configs[i].ConfigName = newCfg.Spec.Name
			}
		}

		if _, err := client.ServiceUpdate(ctx, svc.ID, svc.Version, updated, swarm.ServiceUpdateOptions{}); err != nil {
			return fmt.Errorf("failed to rotate config in service %s: %w", svc.Spec.Name, err)
		}

		l().Infof("Rotated config in service %s to %s", svc.Spec.Name, newCfg.Spec.Name)
	}

	return nil
}

// DeleteConfig deletes a config only if it's not referenced by any service.
func DeleteConfig(ctx context.Context, nameOrID string) error {
	cfg, err := InspectConfig(ctx, nameOrID)
	if err != nil {
		return err
	}

	cli, err := GetClient()
	if err != nil {
		return err
	}
	defer closeCli(cli)

	svcs, err := listServicesUsingConfig(ctx, cli, cfg.Config.ID)
	if err != nil {
		return err
	}
	if len(svcs) > 0 {
		names := make([]string, len(svcs))
		for i, s := range svcs {
			names[i] = s.Spec.Name
		}
		return fmt.Errorf("cannot delete config %q: still used by services %v", cfg.Config.Spec.Name, names)
	}

	return cli.ConfigRemove(ctx, cfg.Config.ID)
}

// --- Helpers ---

var versionSuffix = regexp.MustCompile(`^(.*)-v(\d+)$`)

func nextConfigVersionName(baseName string) string {
	if m := versionSuffix.FindStringSubmatch(baseName); len(m) == 3 {
		prefix := m[1]
		v, _ := strconv.Atoi(m[2])
		return fmt.Sprintf("%s-v%d", prefix, v+1)
	}
	return fmt.Sprintf("%s-v2", baseName)
}

func listServicesUsingConfig(ctx context.Context, client *client.Client, configID string) ([]swarm.Service, error) {
	services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	var filtered []swarm.Service
	for _, s := range services {
		for _, c := range s.Spec.TaskTemplate.ContainerSpec.Configs {
			if c.ConfigID == configID {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered, nil
}

func listServicesUsingConfigName(ctx context.Context, client *client.Client, name string) ([]swarm.Service, error) {
	services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, err
	}
	var filtered []swarm.Service
	for _, s := range services {
		for _, c := range s.Spec.TaskTemplate.ContainerSpec.Configs {
			if c.ConfigName == name {
				filtered = append(filtered, s)
				break
			}
		}
	}
	return filtered, nil
}
