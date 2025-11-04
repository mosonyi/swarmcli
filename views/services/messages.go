package servicesview

type Msg struct {
	Title      string
	Entries    []ServiceEntry
	FilterType FilterType
	NodeID     string
	Hostname   string
	StackName  string
}
