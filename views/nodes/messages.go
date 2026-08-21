// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"time"
)

type Msg struct {
	Entries []docker.NodeEntry
}

// TickMsg triggers periodic node list check
type TickMsg struct{ Gen uint64 }

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

// Poll interval for checking node changes
// PollInterval is how often the view re-reads its resource. It is a var, not a
// const, so tests can shrink it: a tea.Tick cmd invoked synchronously blocks
// for the full interval, so a test that runs one to see what it scheduled would
// otherwise sit here for five seconds.
var PollInterval = 5 * time.Second

// DemoteErrorMsg reports an error occurred while attempting to demote a node.
type DemoteErrorMsg struct {
	NodeID string
	Error  error
}

// PromoteErrorMsg reports an error occurred while attempting to promote a node.
type PromoteErrorMsg struct {
	NodeID string
	Error  error
}

// DemoteSuccessMsg indicates a node was successfully demoted.
type DemoteSuccessMsg struct{}

// PromoteSuccessMsg indicates a node was successfully promoted.
type PromoteSuccessMsg struct{}

// RemoveErrorMsg reports an error occurred while attempting to remove a node.
type RemoveErrorMsg struct {
	NodeID string
	Error  error
}

// RemoveSuccessMsg indicates a node was successfully removed.
type RemoveSuccessMsg struct{}

// SetAvailabilityErrorMsg reports an error occurred while setting node availability.
type SetAvailabilityErrorMsg struct {
	NodeID string
	Error  error
}

// SetAvailabilitySuccessMsg indicates node availability was successfully changed.
type SetAvailabilitySuccessMsg struct{}

// AddLabelErrorMsg reports an error occurred while adding a node label.
type AddLabelErrorMsg struct {
	NodeID string
	Error  error
}

// AddLabelSuccessMsg indicates a label was successfully added to a node.
type AddLabelSuccessMsg struct{}

// RemoveLabelErrorMsg reports an error occurred while removing a node label.
type RemoveLabelErrorMsg struct {
	NodeID string
	Error  error
}

// RemoveLabelSuccessMsg indicates a label was successfully removed from a node.
type RemoveLabelSuccessMsg struct{}
