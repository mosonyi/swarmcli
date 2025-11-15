package servicesview

import "swarmcli/docker"

type Msg struct {
	Title      string
	Entries    []docker.ServiceEntry
	FilterType FilterType
	NodeID     string
	Hostname   string
	StackName  string
}
