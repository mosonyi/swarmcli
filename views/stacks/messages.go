package stacksview

import "swarmcli/docker"

type Msg struct {
	NodeId   string
	Services []docker.StackService
	Error    string
}
