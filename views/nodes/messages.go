package nodesview

import (
	"swarmcli/docker"
	"time"
)

type Msg struct {
	Entries []docker.NodeEntry
}

// TickMsg triggers periodic node list check
type TickMsg time.Time

// Poll interval for checking node changes
const PollInterval = 5 * time.Second

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
