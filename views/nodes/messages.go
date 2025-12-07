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
