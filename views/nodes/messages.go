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
