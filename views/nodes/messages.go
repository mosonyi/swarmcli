package nodesview

import "swarmcli/docker"

type Msg struct {
	Entries []docker.NodeEntry
}
