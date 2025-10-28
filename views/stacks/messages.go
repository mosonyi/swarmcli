package stacksview

import "swarmcli/docker"

type Msg struct {
	NodeId string
	Stacks []docker.Stack
	Error  string
}

type RefreshErrorMsg struct {
	Err error
}
