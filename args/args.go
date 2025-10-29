package args

import "fmt"

// Args holds both positional arguments and flag values.
type Args struct {
	Positionals []string
	Flags       map[string]string
}

// Get returns the string value of a flag or empty string if not present.
func (a *Args) Get(name string) string {
	return a.Flags[name]
}

// Has returns true if a flag was provided.
func (a *Args) Has(name string) bool {
	_, ok := a.Flags[name]
	return ok
}

// String provides a debug-friendly representation.
func (a Args) String() string {
	return fmt.Sprintf("Args{Positionals=%v, Flags=%v}", a.Positionals, a.Flags)
}
