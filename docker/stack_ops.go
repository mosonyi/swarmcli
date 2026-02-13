// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// StackOps abstracts stack operations for testability and extensibility.
type StackOps interface {
	RemoveStack(stackName string) error
	DeployStack(stackName string, yamlContent string) error
	ValidateStackYAML(content string) error
}

type defaultStackOps struct{}

func (defaultStackOps) RemoveStack(stackName string) error {
	return RemoveStack(stackName)
}

func (defaultStackOps) DeployStack(stackName string, yamlContent string) error {
	return DeployStack(stackName, yamlContent)
}

func (defaultStackOps) ValidateStackYAML(content string) error {
	return ValidateStackYAML(content)
}
