// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// Deps aggregates all Docker operation interfaces.
// Views and commands receive this to access Docker operations
// through interfaces rather than package-level functions.
type Deps struct {
	Services    ServiceOps
	Nodes       NodeOps
	Tasks       TaskOps
	Stacks      StackOps
	Configs     ConfigOps
	Secrets     SecretOps
	Networks    NetworkOps
	Contexts    ContextOps
	Snapshot    SnapshotOps
	ClusterInfo ClusterInfoOps
	Inspect     InspectOps
	Events      EventOps
	Client      ClientOps
	Hostname    HostnameOps
}

// DefaultDeps returns a Deps with all default implementations that
// delegate to the existing package-level functions.
func DefaultDeps() Deps {
	return Deps{
		Services:    defaultServiceOps{},
		Nodes:       defaultNodeOps{},
		Tasks:       defaultTaskOps{},
		Stacks:      defaultStackOps{},
		Configs:     defaultConfigOps{},
		Secrets:     defaultSecretOps{},
		Networks:    defaultNetworkOps{},
		Contexts:    defaultContextOps{},
		Snapshot:    defaultSnapshotOps{},
		ClusterInfo: defaultClusterInfoOps{},
		Inspect:     defaultInspectOps{},
		Events:      defaultEventOps{},
		Client:      defaultClientOps{},
		Hostname:    defaultHostnameOps{},
	}
}
