// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/swarm"
)

// ServiceOps abstracts service operations for testability and extensibility.
type ServiceOps interface {
	ScaleService(serviceID string, replicas uint64) error
	ScaleServiceByName(serviceName string, replicas uint64) error
	RestartService(serviceName string) error
	RemoveService(serviceName string) error
	RollbackService(serviceName string) error
	RestartServiceAndWait(ctx context.Context, serviceName string) error
	RestartServiceWithProgress(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error
	LoadNodeServices(nodeID string) []ServiceEntry
	LoadStackServices(stackName string) []ServiceEntry
	GetServiceLogs(ctx context.Context, serviceID string) (string, error)
	GetServiceTaskDiagnostics(ctx context.Context, serviceID string) (string, error)
	CreateService(ctx context.Context, spec swarm.ServiceSpec) (string, error)
	CreateSecretRevealService(serviceName, secretID, secretName string) swarm.ServiceSpec
	CreateSecretRevealServiceWithImage(serviceName, imageOverride, secretID, secretName string) swarm.ServiceSpec
}

type defaultServiceOps struct{}

func (defaultServiceOps) ScaleService(serviceID string, replicas uint64) error {
	return ScaleService(serviceID, replicas)
}
func (defaultServiceOps) ScaleServiceByName(serviceName string, replicas uint64) error {
	return ScaleServiceByName(serviceName, replicas)
}
func (defaultServiceOps) RestartService(serviceName string) error {
	return RestartService(serviceName)
}
func (defaultServiceOps) RemoveService(serviceName string) error {
	return RemoveService(serviceName)
}
func (defaultServiceOps) RollbackService(serviceName string) error {
	return RollbackService(serviceName)
}
func (defaultServiceOps) RestartServiceAndWait(ctx context.Context, serviceName string) error {
	return RestartServiceAndWait(ctx, serviceName)
}
func (defaultServiceOps) RestartServiceWithProgress(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error {
	return RestartServiceWithProgress(ctx, serviceName, progressCh)
}
func (defaultServiceOps) LoadNodeServices(nodeID string) []ServiceEntry {
	return LoadNodeServices(nodeID)
}
func (defaultServiceOps) LoadStackServices(stackName string) []ServiceEntry {
	return LoadStackServices(stackName)
}
func (defaultServiceOps) GetServiceLogs(ctx context.Context, serviceID string) (string, error) {
	return GetServiceLogs(ctx, serviceID)
}
func (defaultServiceOps) GetServiceTaskDiagnostics(ctx context.Context, serviceID string) (string, error) {
	return GetServiceTaskDiagnostics(ctx, serviceID)
}
func (defaultServiceOps) CreateService(ctx context.Context, spec swarm.ServiceSpec) (string, error) {
	return CreateService(ctx, spec)
}
func (defaultServiceOps) CreateSecretRevealService(serviceName, secretID, secretName string) swarm.ServiceSpec {
	return CreateSecretRevealService(serviceName, secretID, secretName)
}
func (defaultServiceOps) CreateSecretRevealServiceWithImage(serviceName, imageOverride, secretID, secretName string) swarm.ServiceSpec {
	return CreateSecretRevealServiceWithImage(serviceName, imageOverride, secretID, secretName)
}
