package docker

type StackService struct {
	NodeID         string
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}
