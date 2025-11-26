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
	RenderItem func(item T, selected bool) string
}

type ModeType int

const (
	ModeNormal ModeType = iota
	ModeSearching
)
