// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

// SnapshotOps abstracts snapshot cache operations for testability and extensibility.
type SnapshotOps interface {
	GetSnapshot() *SwarmSnapshot
	SetSnapshot(s *SwarmSnapshot)
	InvalidateSnapshot()
	RefreshSnapshot() (*SwarmSnapshot, error)
	RefreshSnapshotAsync()
	TriggerRefreshIfNeeded()
	GetOrRefreshSnapshot() (*SwarmSnapshot, error)
}

type defaultSnapshotOps struct{}

func (defaultSnapshotOps) GetSnapshot() *SwarmSnapshot              { return GetSnapshot() }
func (defaultSnapshotOps) SetSnapshot(s *SwarmSnapshot)             { SetSnapshot(s) }
func (defaultSnapshotOps) InvalidateSnapshot()                      { InvalidateSnapshot() }
func (defaultSnapshotOps) RefreshSnapshot() (*SwarmSnapshot, error) { return RefreshSnapshot() }
func (defaultSnapshotOps) RefreshSnapshotAsync()                    { RefreshSnapshotAsync() }
func (defaultSnapshotOps) TriggerRefreshIfNeeded()                  { TriggerRefreshIfNeeded() }
func (defaultSnapshotOps) GetOrRefreshSnapshot() (*SwarmSnapshot, error) {
	return GetOrRefreshSnapshot()
}
