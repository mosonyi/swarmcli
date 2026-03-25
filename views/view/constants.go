// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

// View name constants for type-safe navigation
const (
	NameHelp     = "help"
	NameStacks   = "stacks"
	NameServices = "services"
	NameTasks    = "tasks"
	NameNodes    = "nodes"
	NameConfigs  = "configs"
	NameSecrets  = "secrets"
	NameContexts = "contexts"
	NameInspect  = "inspect"
	NameLogs     = "logs"
	NameLoading  = "loading"
	NameNetworks = "networks"
)

var topLevelViews = map[string]bool{
	NameStacks: true, NameNodes: true, NameConfigs: true,
	NameSecrets: true, NameNetworks: true, NameContexts: true,
	NameLoading: true, NameHelp: true,
}

// RegisterTopLevel marks a view name as top-level. Called from init() by
// extension modules to register additional root views.
func RegisterTopLevel(name string) { topLevelViews[name] = true }

// IsTopLevel reports whether a view name is a top-level (root) view.
func IsTopLevel(name string) bool { return topLevelViews[name] }
