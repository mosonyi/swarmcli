package view

type NavigateToMsg struct {
	ViewName string
	Payload  any // Can be service ID, stack ID, etc.
}

type NavigateBackMsg struct{}
