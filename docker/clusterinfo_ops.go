// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// ClusterInfoOps abstracts cluster info queries for testability and extensibility.
type ClusterInfoOps interface {
	GetCurrentContext() (string, error)
	GetContainerCount() (int, error)
	GetServiceCount() (int, error)
	GetSwarmCPUCapacity() (float64, error)
	GetSwarmMemCapacity() (int64, error)
	GetSwarmResourceUsage() (cpuPct, memPct string, err error)
	GetDockerVersion() (string, error)
}

type defaultClusterInfoOps struct{}

func (defaultClusterInfoOps) GetCurrentContext() (string, error)    { return GetCurrentContext() }
func (defaultClusterInfoOps) GetContainerCount() (int, error)       { return GetContainerCount() }
func (defaultClusterInfoOps) GetServiceCount() (int, error)         { return GetServiceCount() }
func (defaultClusterInfoOps) GetSwarmCPUCapacity() (float64, error) { return GetSwarmCPUCapacity() }
func (defaultClusterInfoOps) GetSwarmMemCapacity() (int64, error)   { return GetSwarmMemCapacity() }
func (defaultClusterInfoOps) GetSwarmResourceUsage() (string, string, error) {
	return GetSwarmResourceUsage()
}
func (defaultClusterInfoOps) GetDockerVersion() (string, error) { return GetDockerVersion() }
