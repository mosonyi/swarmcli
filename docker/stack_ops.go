// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// StackOps abstracts stack operations for testability and extensibility.
type StackOps interface {
	RemoveStack(stackName string) error
}

type defaultStackOps struct{}

func (defaultStackOps) RemoveStack(stackName string) error {
	return RemoveStack(stackName)
}
