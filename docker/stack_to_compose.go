// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeFile represents a Docker Compose file structure
type ComposeFile struct {
	Version  string                    `yaml:"version"`
	Services map[string]ComposeService `yaml:"services"`
	Networks map[string]map[string]any `yaml:"networks,omitempty"`
	Volumes  map[string]map[string]any `yaml:"volumes,omitempty"`
	Secrets  map[string]map[string]any `yaml:"secrets,omitempty"`
	Configs  map[string]map[string]any `yaml:"configs,omitempty"`
}

// ComposeService represents a service in a Docker Compose file
type ComposeService struct {
	Image       string            `yaml:"image,omitempty"`
	Command     any               `yaml:"command,omitempty"` // string or []string
	Entrypoint  any               `yaml:"entrypoint,omitempty"`
	WorkingDir  string            `yaml:"working_dir,omitempty"`
	User        string            `yaml:"user,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Ports       []string          `yaml:"ports,omitempty"`
	Networks    any               `yaml:"networks,omitempty"` // []string or map
	Volumes     []string          `yaml:"volumes,omitempty"`
	Secrets     []map[string]any  `yaml:"secrets,omitempty"`
	Configs     []map[string]any  `yaml:"configs,omitempty"`
	Deploy      map[string]any    `yaml:"deploy,omitempty"`
	Extra       map[string]any    `yaml:",inline,omitempty"` // fallback
}

// ServiceInspect represents Docker service inspect output (partial)
type ServiceInspect struct {
	Spec ServiceSpec `json:"Spec"`
}

// ServiceSpec represents the service specification
type ServiceSpec struct {
	Name         string            `json:"Name"`
	Labels       map[string]string `json:"Labels"`
	TaskTemplate TaskTemplate      `json:"TaskTemplate"`
	Mode         ServiceMode       `json:"Mode"`
	Networks     []NetRef          `json:"Networks"`
	EndpointSpec *EndpointSpec     `json:"EndpointSpec,omitempty"`
}

// ServiceMode represents the service mode (replicated or global)
type ServiceMode struct {
	Replicated *struct {
		Replicas *uint64 `json:"Replicas"`
	} `json:"Replicated,omitempty"`
	Global any `json:"Global,omitempty"`
}

// TaskTemplate represents the task template specification
type TaskTemplate struct {
	ContainerSpec *ContainerSpec `json:"ContainerSpec,omitempty"`
	Resources     *Resources     `json:"Resources,omitempty"`
	RestartPolicy *RestartPolicy `json:"RestartPolicy,omitempty"`
	Placement     *Placement     `json:"Placement,omitempty"`
	Networks      []NetRef       `json:"Networks,omitempty"`
	ForceUpdate   uint64         `json:"ForceUpdate,omitempty"`
	LogDriver     any            `json:"LogDriver,omitempty"`
}

// ContainerSpec represents the container specification
type ContainerSpec struct {
	Image   string            `json:"Image"`
	Args    []string          `json:"Args,omitempty"`
	Command []string          `json:"Command,omitempty"`
	Env     []string          `json:"Env,omitempty"`
	Dir     string            `json:"Dir,omitempty"`
	User    string            `json:"User,omitempty"`
	Labels  map[string]string `json:"Labels,omitempty"`
	Mounts  []Mount           `json:"Mounts,omitempty"`
	Secrets []SecretRef       `json:"Secrets,omitempty"`
	Configs []ConfigRef       `json:"Configs,omitempty"`
}

// Mount represents a mount specification
type Mount struct {
	Type        string `json:"Type"` // bind, volume, tmpfs
	Source      string `json:"Source,omitempty"`
	Target      string `json:"Target,omitempty"`
	ReadOnly    bool   `json:"ReadOnly,omitempty"`
	BindOptions *struct {
		Propagation string `json:"Propagation,omitempty"`
	} `json:"BindOptions,omitempty"`
	VolumeOptions *struct {
		NoCopy       bool              `json:"NoCopy,omitempty"`
		Labels       map[string]string `json:"Labels,omitempty"`
		DriverConfig *struct {
			Name    string            `json:"Name,omitempty"`
			Options map[string]string `json:"Options,omitempty"`
		} `json:"DriverConfig,omitempty"`
	} `json:"VolumeOptions,omitempty"`
	TmpfsOptions *struct {
		SizeBytes int64  `json:"SizeBytes,omitempty"`
		Mode      uint32 `json:"Mode,omitempty"`
	} `json:"TmpfsOptions,omitempty"`
}

// SecretRef represents a secret reference
type SecretRef struct {
	SecretID   string `json:"SecretID"`
	SecretName string `json:"SecretName"`
	File       *struct {
		Name string `json:"Name"`
		UID  string `json:"UID,omitempty"`
		GID  string `json:"GID,omitempty"`
		Mode uint32 `json:"Mode,omitempty"`
	} `json:"File,omitempty"`
}

// ConfigRef represents a config reference
type ConfigRef struct {
	ConfigID   string `json:"ConfigID"`
	ConfigName string `json:"ConfigName"`
	File       *struct {
		Name string `json:"Name"`
		UID  string `json:"UID,omitempty"`
		GID  string `json:"GID,omitempty"`
		Mode uint32 `json:"Mode,omitempty"`
	} `json:"File,omitempty"`
}

// NetRef represents a network reference
type NetRef struct {
	Target  string   `json:"Target"` // network ID
	Aliases []string `json:"Aliases,omitempty"`
}

// Resources represents resource constraints
type Resources struct {
	Limits       *ResourceSpec `json:"Limits,omitempty"`
	Reservations *ResourceSpec `json:"Reservations,omitempty"`
}

// ResourceSpec represents resource specifications
type ResourceSpec struct {
	NanoCPUs    int64 `json:"NanoCPUs,omitempty"`
	MemoryBytes int64 `json:"MemoryBytes,omitempty"`
}

// RestartPolicy represents the restart policy
type RestartPolicy struct {
	Condition   string  `json:"Condition,omitempty"`
	Delay       int64   `json:"Delay,omitempty"`
	MaxAttempts *uint64 `json:"MaxAttempts,omitempty"`
	Window      int64   `json:"Window,omitempty"`
}

// Placement represents placement constraints
type Placement struct {
	Constraints []string `json:"Constraints,omitempty"`
	Preferences []any    `json:"Preferences,omitempty"`
	MaxReplicas *uint64  `json:"MaxReplicas,omitempty"`
}

// EndpointSpec represents the endpoint specification
type EndpointSpec struct {
	Ports []PortConfig `json:"Ports,omitempty"`
}

// PortConfig represents a port configuration
type PortConfig struct {
	Protocol      string `json:"Protocol,omitempty"` // tcp/udp
	TargetPort    uint32 `json:"TargetPort,omitempty"`
	PublishedPort uint32 `json:"PublishedPort,omitempty"`
	PublishMode   string `json:"PublishMode,omitempty"` // ingress/host
}

// ReconstructStackCompose reconstructs a Docker Compose file from a running stack
func ReconstructStackCompose(stackName string) (string, error) {
	// Get list of services in the stack
	serviceNames, err := getStackServices(stackName)
	if err != nil {
		return "", fmt.Errorf("failed to list services: %w", err)
	}

	if len(serviceNames) == 0 {
		return "", fmt.Errorf("no services found in stack %q", stackName)
	}

	// Build network ID to name mapping
	netID2Name, err := dockerNetworkIDToNameMap()
	if err != nil {
		l().Warnf("Could not build network ID->name map: %v", err)
		netID2Name = map[string]string{}
	}

	// Initialize compose file structure
	cf := ComposeFile{
		Version:  "3.8",
		Services: map[string]ComposeService{},
		Networks: map[string]map[string]any{},
		Volumes:  map[string]map[string]any{},
		Secrets:  map[string]map[string]any{},
		Configs:  map[string]map[string]any{},
	}

	// Helper functions for declaring resources
	declareExternalNet := func(netName string) {
		if netName == "" {
			return
		}
		if _, ok := cf.Networks[netName]; !ok {
			cf.Networks[netName] = map[string]any{"external": true}
		}
	}

	declareVolume := func(volName string, m *Mount, external bool) {
		if volName == "" {
			return
		}
		if _, ok := cf.Volumes[volName]; ok {
			return
		}
		vol := map[string]any{}
		if external {
			vol["external"] = true
		}
		if m != nil && m.VolumeOptions != nil && m.VolumeOptions.DriverConfig != nil {
			dc := m.VolumeOptions.DriverConfig
			if dc.Name != "" {
				vol["driver"] = dc.Name
				delete(vol, "external")
			}
			if len(dc.Options) > 0 {
				vol["driver_opts"] = dc.Options
			}
		}
		cf.Volumes[volName] = vol
	}

	declareSecretExternal := func(name string) {
		if name == "" {
			return
		}
		if _, ok := cf.Secrets[name]; !ok {
			cf.Secrets[name] = map[string]any{"external": true}
		}
	}

	declareConfigExternal := func(name string) {
		if name == "" {
			return
		}
		if _, ok := cf.Configs[name]; !ok {
			cf.Configs[name] = map[string]any{"external": true}
		}
	}

	// Process each service
	for _, fullSvcName := range serviceNames {
		si, err := inspectService(fullSvcName)
		if err != nil {
			l().Warnf("Failed to inspect service %s: %v", fullSvcName, err)
			continue
		}

		key := stripStackPrefix(stackName, si.Spec.Name)
		cs := ComposeService{}

		// ServiceSpec.Labels → deploy.labels in Compose
		deployLabels := filterLabels(si.Spec.Labels)

		if si.Spec.TaskTemplate.ContainerSpec != nil {
			cspec := si.Spec.TaskTemplate.ContainerSpec
			cs.Image = cspec.Image

			if cspec.Dir != "" {
				cs.WorkingDir = cspec.Dir
			}
			if cspec.User != "" {
				cs.User = cspec.User
			}
			if env := parseKeyValEnv(cspec.Env); env != nil {
				cs.Environment = env
			}

			// ContainerSpec.Labels → service-level labels in Compose
			if cl := filterLabels(cspec.Labels); len(cl) > 0 {
				cs.Labels = cl
			}

			// Command / Args
			if len(cspec.Args) > 0 {
				cs.Command = cspec.Args
			} else if len(cspec.Command) > 0 {
				cs.Command = cspec.Command
			}

			// Mounts -> volumes
			for _, m := range cspec.Mounts {
				if m.Target == "" {
					continue
				}
				ro := ""
				if m.ReadOnly {
					ro = ":ro"
				}
				switch m.Type {
				case "bind":
					if m.Source != "" {
						cs.Volumes = append(cs.Volumes, fmt.Sprintf("%s:%s%s", m.Source, m.Target, ro))
					}
				case "volume":
					src := m.Source
					if src == "" {
						continue // anonymous volume, cannot round-trip
					}
					stripped := stripStackPrefix(stackName, src)
					isExternal := stripped == src // no prefix removed → external
					declareVolume(stripped, &m, isExternal)
					cs.Volumes = append(cs.Volumes, fmt.Sprintf("%s:%s%s", stripped, m.Target, ro))
				case "tmpfs":
					if cs.Extra == nil {
						cs.Extra = map[string]any{}
					}
					tmpfs, _ := cs.Extra["tmpfs"].([]any)
					tmpfs = append(tmpfs, m.Target)
					cs.Extra["tmpfs"] = tmpfs
				}
			}

			// Secrets
			for _, s := range cspec.Secrets {
				if s.SecretName == "" {
					continue
				}
				declareSecretExternal(s.SecretName)
				ref := map[string]any{"source": s.SecretName}
				if s.File != nil && s.File.Name != "" {
					ref["target"] = s.File.Name
				}
				cs.Secrets = append(cs.Secrets, ref)
			}

			// Configs
			for _, c := range cspec.Configs {
				if c.ConfigName == "" {
					continue
				}
				declareConfigExternal(c.ConfigName)
				ref := map[string]any{"source": c.ConfigName}
				if c.File != nil && c.File.Name != "" {
					ref["target"] = c.File.Name
				}
				cs.Configs = append(cs.Configs, ref)
			}
		}

		// Ports
		if si.Spec.EndpointSpec != nil {
			for _, p := range si.Spec.EndpointSpec.Ports {
				if p.TargetPort == 0 {
					continue
				}
				proto := p.Protocol
				if proto == "" {
					proto = "tcp"
				}
				if p.PublishedPort != 0 {
					cs.Ports = append(cs.Ports, fmt.Sprintf("%d:%d/%s", p.PublishedPort, p.TargetPort, proto))
				} else {
					cs.Ports = append(cs.Ports, fmt.Sprintf("%d/%s", p.TargetPort, proto))
				}
			}
		}

		// Networks
		netRefs := append([]NetRef{}, si.Spec.TaskTemplate.Networks...)
		netRefs = append(netRefs, si.Spec.Networks...)
		netNames := make(map[string]struct{})
		for _, nr := range netRefs {
			nm := netID2Name[nr.Target]
			if nm == "" {
				nm = nr.Target
			}
			if nm == "" {
				continue
			}
			nm = stripStackPrefix(stackName, nm)
			netNames[nm] = struct{}{}
			declareExternalNet(nm)
		}
		if len(netNames) > 0 {
			var nets []string
			for n := range netNames {
				nets = append(nets, n)
			}
			sort.Strings(nets)
			cs.Networks = nets
		}

		// Deploy section
		deploy := make(map[string]any)

		// Mode/replicas
		if si.Spec.Mode.Replicated != nil && si.Spec.Mode.Replicated.Replicas != nil {
			deploy["replicas"] = int(*si.Spec.Mode.Replicated.Replicas)
		} else if si.Spec.Mode.Global != nil {
			deploy["mode"] = "global"
		}

		// Placement constraints
		if si.Spec.TaskTemplate.Placement != nil && len(si.Spec.TaskTemplate.Placement.Constraints) > 0 {
			deploy["placement"] = map[string]any{
				"constraints": si.Spec.TaskTemplate.Placement.Constraints,
			}
		}

		// Resources
		if si.Spec.TaskTemplate.Resources != nil {
			res := make(map[string]any)
			if si.Spec.TaskTemplate.Resources.Limits != nil {
				lim := make(map[string]any)
				if si.Spec.TaskTemplate.Resources.Limits.NanoCPUs != 0 {
					lim["cpus"] = nanoCPUToCPUString(si.Spec.TaskTemplate.Resources.Limits.NanoCPUs)
				}
				if si.Spec.TaskTemplate.Resources.Limits.MemoryBytes != 0 {
					lim["memory"] = bytesToHuman(si.Spec.TaskTemplate.Resources.Limits.MemoryBytes)
				}
				if len(lim) > 0 {
					res["limits"] = lim
				}
			}
			if si.Spec.TaskTemplate.Resources.Reservations != nil {
				resv := make(map[string]any)
				if si.Spec.TaskTemplate.Resources.Reservations.NanoCPUs != 0 {
					resv["cpus"] = nanoCPUToCPUString(si.Spec.TaskTemplate.Resources.Reservations.NanoCPUs)
				}
				if si.Spec.TaskTemplate.Resources.Reservations.MemoryBytes != 0 {
					resv["memory"] = bytesToHuman(si.Spec.TaskTemplate.Resources.Reservations.MemoryBytes)
				}
				if len(resv) > 0 {
					res["reservations"] = resv
				}
			}
			if len(res) > 0 {
				deploy["resources"] = res
			}
		}

		// Restart policy
		if si.Spec.TaskTemplate.RestartPolicy != nil {
			rp := make(map[string]any)
			if si.Spec.TaskTemplate.RestartPolicy.Condition != "" {
				rp["condition"] = si.Spec.TaskTemplate.RestartPolicy.Condition
			}
			if si.Spec.TaskTemplate.RestartPolicy.MaxAttempts != nil {
				rp["max_attempts"] = int(*si.Spec.TaskTemplate.RestartPolicy.MaxAttempts)
			}
			if len(rp) > 0 {
				deploy["restart_policy"] = rp
			}
		}

		if len(deployLabels) > 0 {
			deploy["labels"] = deployLabels
		}

		if len(deploy) > 0 {
			cs.Deploy = deploy
		}

		cf.Services[key] = cs
	}

	// If the only declared network is "default" (implicit), suppress it
	if len(cf.Networks) == 1 {
		if _, ok := cf.Networks["default"]; ok {
			cf.Networks = nil
			for k, svc := range cf.Services {
				if nets, ok := svc.Networks.([]string); ok && len(nets) == 1 && nets[0] == "default" {
					svc.Networks = nil
					cf.Services[k] = svc
				}
			}
		}
	}

	// Remove empty top-level sections
	if len(cf.Networks) == 0 {
		cf.Networks = nil
	}
	if len(cf.Volumes) == 0 {
		cf.Volumes = nil
	}
	if len(cf.Secrets) == 0 {
		cf.Secrets = nil
	}
	if len(cf.Configs) == 0 {
		cf.Configs = nil
	}

	// Marshal to YAML
	y, err := yaml.Marshal(&cf)
	if err != nil {
		return "", fmt.Errorf("yaml marshal error: %w", err)
	}

	return string(y), nil
}

// getStackServices returns the list of service names for a stack
func getStackServices(stackName string) ([]string, error) {
	cmd := exec.Command("docker", "stack", "services", stackName, "--format", "{{.Name}}")
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("docker stack services failed: %v\n%s", err, errb.String())
	}

	var serviceNames []string
	for _, ln := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			serviceNames = append(serviceNames, ln)
		}
	}
	sort.Strings(serviceNames)
	return serviceNames, nil
}

// inspectService returns the inspect data for a service
func inspectService(serviceName string) (*ServiceInspect, error) {
	cmd := exec.Command("docker", "service", "inspect", serviceName)
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("docker service inspect failed: %v\n%s", err, errb.String())
	}

	var arr []ServiceInspect
	if err := json.Unmarshal(out.Bytes(), &arr); err != nil || len(arr) == 0 {
		return nil, fmt.Errorf("failed to parse service inspect json: %w", err)
	}
	return &arr[0], nil
}

// dockerNetworkIDToNameMap builds a map of network IDs to names
func dockerNetworkIDToNameMap() (map[string]string, error) {
	cmd := exec.Command("docker", "network", "ls", "--no-trunc", "--format", "{{.ID}}\t{{.Name}}")
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("docker network ls failed: %v\n%s", err, errb.String())
	}

	m := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		parts := strings.SplitN(ln, "\t", 2)
		if len(parts) == 2 {
			m[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return m, nil
}

// parseKeyValEnv parses environment variables in KEY=VALUE format
func parseKeyValEnv(env []string) map[string]string {
	m := make(map[string]string)
	for _, kv := range env {
		if kv == "" {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 1 {
			m[parts[0]] = ""
		} else {
			m[parts[0]] = parts[1]
		}
	}
	if len(m) == 0 {
		return nil
	}
	return m
}

// filterLabels returns a copy of labels with Docker-internal keys removed.
// Returns nil if no user labels remain.
func filterLabels(labels map[string]string) map[string]string {
	out := make(map[string]string)
	for k, v := range labels {
		if !strings.HasPrefix(k, "com.docker.stack.") {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// stripStackPrefix removes the "<stack>_" prefix from a resource name
func stripStackPrefix(stack, full string) string {
	prefix := stack + "_"
	if strings.HasPrefix(full, prefix) {
		return strings.TrimPrefix(full, prefix)
	}
	return full
}

// nanoCPUToCPUString converts nano CPUs to CPU string
func nanoCPUToCPUString(n int64) string {
	f := float64(n) / 1e9
	s := strconv.FormatFloat(f, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

// bytesToHuman converts bytes to human-readable format
func bytesToHuman(b int64) string {
	abs := b
	if abs < 0 {
		abs = -abs
	}
	const (
		KiB = 1024
		MiB = 1024 * KiB
		GiB = 1024 * MiB
	)
	switch {
	case abs >= GiB && abs%GiB == 0:
		return fmt.Sprintf("%dG", b/GiB)
	case abs >= MiB && abs%MiB == 0:
		return fmt.Sprintf("%dM", b/MiB)
	case abs >= KiB && abs%KiB == 0:
		return fmt.Sprintf("%dK", b/KiB)
	default:
		return fmt.Sprintf("%d", b)
	}
}
