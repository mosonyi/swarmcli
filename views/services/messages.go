package servicesview

import (
	"swarmcli/docker"
	"time"
)

type Msg struct {
	Title      string
	Entries    []docker.ServiceEntry
	FilterType FilterType
	NodeID     string
	Hostname   string
	StackName  string
}

type TickMsg time.Time

const PollInterval = 5 * time.Second

// RestartErrorMsg is sent when a service restart fails
type RestartErrorMsg struct {
	ServiceName string
	Error       error
}

// ScaleErrorMsg is sent when a service scale operation fails
type ScaleErrorMsg struct {
	ServiceName string
	Error       error
}
