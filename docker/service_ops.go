// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/swarm"
)

// ServiceOps abstracts service operations for testability and extensibility.
type ServiceOps interface {
	ScaleService(ctx context.Context, serviceID string, replicas uint64) error
	ScaleServiceByName(ctx context.Context, serviceName string, replicas uint64) error
	RestartService(ctx context.Context, serviceName string) error
	RemoveService(ctx context.Context, serviceName string) error
	RollbackService(ctx context.Context, serviceName string) error
	RestartServiceAndWait(ctx context.Context, serviceName string) error
	RestartServiceWithProgress(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error
	LoadNodeServices(nodeID string) []ServiceEntry
	LoadStackServices(stackName string) []ServiceEntry
	GetServiceLogs(ctx context.Context, serviceID string) (string, error)
	GetServiceTaskDiagnostics(ctx context.Context, serviceID string) (string, error)
	CreateService(ctx context.Context, spec swarm.ServiceSpec) (string, error)
}

type defaultServiceOps struct{}

func (defaultServiceOps) ScaleService(ctx context.Context, serviceID string, replicas uint64) error {
	return ScaleService(ctx, serviceID, replicas)
}
func (defaultServiceOps) ScaleServiceByName(ctx context.Context, serviceName string, replicas uint64) error {
	return ScaleServiceByName(ctx, serviceName, replicas)
}
func (defaultServiceOps) RestartService(ctx context.Context, serviceName string) error {
	return RestartService(ctx, serviceName)
}
func (defaultServiceOps) RemoveService(ctx context.Context, serviceName string) error {
	return RemoveService(ctx, serviceName)
}
func (defaultServiceOps) RollbackService(ctx context.Context, serviceName string) error {
	return RollbackService(ctx, serviceName)
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
