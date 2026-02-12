// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// TaskOps abstracts task query operations for testability and extensibility.
type TaskOps interface {
	GetTasksForStack(stackName string) ([]TaskEntry, error)
	GetTasksForService(serviceID string) ([]TaskEntry, error)
}

type defaultTaskOps struct{}

func (defaultTaskOps) GetTasksForStack(stackName string) ([]TaskEntry, error) {
	return GetTasksForStack(stackName)
}
func (defaultTaskOps) GetTasksForService(serviceID string) ([]TaskEntry, error) {
	return GetTasksForService(serviceID)
}
