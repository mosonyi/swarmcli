// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/swarm"
)

// ConfigOps abstracts config operations for testability and extensibility.
type ConfigOps interface {
	ListConfigs(ctx context.Context) ([]swarm.Config, error)
	InspectConfig(ctx context.Context, nameOrID string) (*ConfigWithDecodedData, error)
	CreateConfig(ctx context.Context, name string, data []byte, labels map[string]string) (swarm.Config, error)
	CreateConfigVersion(ctx context.Context, baseConfig swarm.Config, newData []byte) (swarm.Config, error)
	RotateConfigInServices(ctx context.Context, oldCfg *swarm.Config, newCfg swarm.Config) error
	DeleteConfig(ctx context.Context, nameOrID string) error
	// ServicesUsingConfigs answers "which services reference this config" for
	// every config at once, keyed by config ID and name.
	//
	// It replaces per-config lookups in this seam. ListServicesUsingConfigID and
	// ListServicesUsingConfigName still exist on the package for a caller with a
	// single config, but each lists every service and filters, so a loop over
	// them scales the whole service listing by the number of configs.
	ServicesUsingConfigs(ctx context.Context) (map[string][]swarm.Service, error)
}

type defaultConfigOps struct{}

func (defaultConfigOps) ListConfigs(ctx context.Context) ([]swarm.Config, error) {
	return ListConfigs(ctx)
}
func (defaultConfigOps) InspectConfig(ctx context.Context, nameOrID string) (*ConfigWithDecodedData, error) {
	return InspectConfig(ctx, nameOrID)
}
func (defaultConfigOps) CreateConfig(ctx context.Context, name string, data []byte, labels map[string]string) (swarm.Config, error) {
	return CreateConfig(ctx, name, data, labels)
}
func (defaultConfigOps) CreateConfigVersion(ctx context.Context, baseConfig swarm.Config, newData []byte) (swarm.Config, error) {
	return CreateConfigVersion(ctx, baseConfig, newData)
}
func (defaultConfigOps) RotateConfigInServices(ctx context.Context, oldCfg *swarm.Config, newCfg swarm.Config) error {
	return RotateConfigInServices(ctx, oldCfg, newCfg)
}
func (defaultConfigOps) DeleteConfig(ctx context.Context, nameOrID string) error {
	return DeleteConfig(ctx, nameOrID)
}
func (defaultConfigOps) ServicesUsingConfigs(ctx context.Context) (map[string][]swarm.Service, error) {
	return ServicesUsingConfigs(ctx)
}
