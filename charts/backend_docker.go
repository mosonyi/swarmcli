// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"

	"swarmcli/docker"
)

// dockerBackend implements Backend against the live swarmcli/docker package.
type dockerBackend struct{}

func (dockerBackend) DeployStack(name, manifest string) error {
	return docker.DeployStack(name, manifest)
}

func (dockerBackend) RemoveStack(name string) error {
	return docker.RemoveStackCLI(name)
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
	out := make([]ServiceState, 0, len(entries))
	for _, e := range entries {
		replicas := ""
		if e.Mode == "replicated" {
			replicas = fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		}
		out = append(out, ServiceState{
			Name:     e.ServiceName,
			Mode:     e.Mode,
			Replicas: replicas,
			Status:   e.Status,
		})
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
