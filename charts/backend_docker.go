// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"

	"github.com/Eldara-Tech/swarmcli/docker"
)

// dockerBackend implements Backend against the live github.com/Eldara-Tech/swarmcli/docker package.
type dockerBackend struct{}

func (dockerBackend) DeployStack(name, manifest string) error {
	return docker.DeployStack(name, manifest)
}

func (dockerBackend) RemoveStack(name string) error {
	return docker.RemoveStackCLI(name)
}

func (dockerBackend) RefreshSnapshot() error {
	_, err := docker.RefreshSnapshot()
	return err
}

func (dockerBackend) CreateConfig(ctx context.Context, name string, data []byte, labels map[string]string) error {
	_, err := docker.CreateConfig(ctx, name, data, labels)
	return err
}

func (dockerBackend) ListConfigs(ctx context.Context) ([]ConfigMeta, error) {
	configs, err := docker.ListConfigs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ConfigMeta, 0, len(configs))
	for _, c := range configs {
		out = append(out, ConfigMeta{Name: c.Spec.Name, Labels: c.Spec.Labels})
	}
	return out, nil
}

func (dockerBackend) InspectConfig(ctx context.Context, name string) ([]byte, error) {
	cfg, err := docker.InspectConfig(ctx, name)
	if err != nil {
		return nil, err
	}
	return cfg.Data, nil
}

func (dockerBackend) DeleteConfig(ctx context.Context, name string) error {
	return docker.DeleteConfig(ctx, name)
}

func (dockerBackend) StackServices(name string) []ServiceState {
	entries := docker.LoadStackServices(name)
	// Convergence facts come from a separate loader because ServiceEntry counts
	// replicas by desired state, which is right for the services view but wrong
	// for deciding a rollout finished (issues #473, #480).
	conv := make(map[string]docker.ServiceConvergence, len(entries))
	for _, c := range docker.LoadStackConvergence(name) {
		conv[c.Name] = c
	}

	out := make([]ServiceState, 0, len(entries))
	for _, e := range entries {
		replicas := ""
		if e.Mode == "replicated" {
			replicas = fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		}
		st := ServiceState{
			Name:     e.ServiceName,
			Mode:     e.Mode,
			Replicas: replicas,
			Status:   e.Status,
		}
		if c, ok := conv[e.ServiceName]; ok {
			st.Running = c.Running
			st.Desired = c.Desired
			st.UpdateState = c.UpdateState
			st.Monitor = c.Monitor
		}
		out = append(out, st)
	}
	return out
}

func (dockerBackend) StackVolumes(ctx context.Context, name string) ([]string, error) {
	vols, err := docker.ListVolumes(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range vols {
		if v.Stack == name {
			out = append(out, v.Name)
		}
	}
	return out, nil
}

func (dockerBackend) RemoveVolume(ctx context.Context, name string) error {
	return docker.RemoveVolume(ctx, name, false)
}

func (dockerBackend) NetworkScopes(ctx context.Context) (map[string]string, error) {
	nets, err := docker.ListNetworks(ctx)
	if err != nil {
		return nil, err
	}
	scopes := make(map[string]string, len(nets))
	for _, n := range nets {
		scopes[n.Name] = n.Scope
	}
	return scopes, nil
}

func (dockerBackend) CreateOverlayNetwork(ctx context.Context, name, driver string, attachable bool) error {
	_, _, err := docker.CreateNetwork(ctx, name, network.CreateOptions{Driver: driver, Attachable: attachable})
	return err
}

func (dockerBackend) RemoveOverlayNetwork(ctx context.Context, name string) error {
	nets, err := docker.ListNetworks(ctx)
	if err != nil {
		return err
	}
	for _, n := range nets {
		if n.Name == name {
			return docker.RemoveNetwork(ctx, n.ID)
		}
	}
	return nil // already gone
}

func (dockerBackend) SecretNames(ctx context.Context) (map[string]struct{}, error) {
	secs, err := docker.ListSecrets(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[string]struct{}, len(secs))
	for _, s := range secs {
		names[s.Spec.Name] = struct{}{}
	}
	return names, nil
}
