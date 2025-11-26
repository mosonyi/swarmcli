package filterlist

import (
	"github.com/charmbracelet/bubbles/viewport"
)

type FilterableList[T any] struct {
	Viewport viewport.Model

	Items    []T
	Filtered []T
	Cursor   int
	Query    string
	Mode     ModeType

	// Function to render a single item
	RenderItem func(item T, selected bool, colWidth int) string

	// Match function for filtering
	Match func(item T, query string) bool

	// Computed column width
	colWidth int
}

type ModeType int

const (
	ModeNormal ModeType = iota
	ModeSearching
)
