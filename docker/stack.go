package docker

// Stack represents a unique Docker stack.
type Stack struct {
	Name         string
	ServiceCount int
}
