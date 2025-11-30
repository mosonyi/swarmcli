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
