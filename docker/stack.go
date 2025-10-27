package docker

type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return GetAllStacks()
	}
	return GetNodeStacks(nodeID)
}
