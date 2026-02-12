// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// HostnameOps abstracts hostname cache operations for testability and extensibility.
type HostnameOps interface {
	RefreshHostnameCache() error
	GetNodeIDToHostnameMap() (map[string]string, error)
}

type defaultHostnameOps struct{}

func (defaultHostnameOps) RefreshHostnameCache() error {
	return RefreshHostnameCache()
}
func (defaultHostnameOps) GetNodeIDToHostnameMap() (map[string]string, error) {
	return GetNodeIDToHostnameMap()
}
