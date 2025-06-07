package stacksview

type Msg struct {
	NodeId   string
	Services []StackService
	Error    string
}
