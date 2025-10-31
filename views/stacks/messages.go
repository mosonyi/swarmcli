package stacksview

import "swarmcli/docker"

type Msg struct {
	NodeID string
	Stacks []docker.StackEntry
	Err    error
}

type RefreshErrorMsg struct {
	Err error
}
