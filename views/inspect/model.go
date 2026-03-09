// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Node struct {
	Key      string
	Raw      any
	ValueStr string
	Children []*Node
	Parent   *Node
}

type Format string

const (
	FormatYAML Format = "yml"
	FormatRaw  Format = "raw"
)

type Model struct {
	viewport   viewport.Model
	Root       *Node
	Title      string
	SearchTerm string
	searchMode bool
	ready      bool
	width      int
	height     int

	Format     Format // "yml" or "raw"
	RawContent string
	ParseError string

	// app-level "/" filter — hides non-matching lines
	filterQuery   string
	searchMatches []int
	searchIndex   int
}

func New(width, height int, format Format) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
		Format:   format,
	}
}

func (m *Model) SetFormat(format Format) {
	m.Format = format

	if m.Format == FormatRaw {
		m.viewport.SetContent(m.RawContent)
	} else {
		m.updateViewport()
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string { return ViewName }

func (m *Model) SetTitle(t string) { m.Title = t }

// LoadInspectItem returns a cmd that sends a Msg(title, json)
func LoadInspectItem(title, jsonStr string) tea.Cmd {
	return func() tea.Msg { return Msg{Title: title, Content: jsonStr} }
}

// ApplySearchQuery implements view.Filterable — sets the app-level "/" filter.
func (m *Model) ApplySearchQuery(query string) {
	m.filterQuery = query
	m.updateViewport()
}

// ClearSearchQuery implements view.Filterable — clears the app-level "/" filter.
func (m *Model) ClearSearchQuery() {
	m.filterQuery = ""
	m.updateViewport()
}

// IsSearching returns true when the ctrl+f search input is active.
func (m *Model) IsSearching() bool {
	return m.searchMode
}

// HasActiveFilter returns true when the app-level "/" filter is active.
func (m *Model) HasActiveFilter() bool {
	return m.filterQuery != ""
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.searchMode {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "Apply"},
			{Key: "esc", Desc: "Cancel"},
		}
	}
	entries := []helpbar.HelpEntry{
		{Key: "ctrl+f", Desc: "Search"},
		{Key: "j/k", Desc: "Down/up"},
	}
	if m.SearchTerm != "" && len(m.searchMatches) > 0 {
		entries = append(entries, helpbar.HelpEntry{Key: "n/N", Desc: "Next/prev"})
	}
	entries = append(entries,
		helpbar.HelpEntry{Key: "r", Desc: "Toggle raw"},
		helpbar.HelpEntry{Key: "q", Desc: "Close"},
	)
	return entries
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

func (m *Model) HasErrors() bool {
	return false
}

func ParseFormat(v any) Format {
	switch x := v.(type) {
	case Format:
		return x
	case string:
		f := Format(x)
		if f == FormatYAML || f == FormatRaw {
			return f
		}
	}
	return FormatYAML
}
