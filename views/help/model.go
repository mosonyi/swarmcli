// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"fmt"
	"strings"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Help view provides a generic categorized help screen for any view.
//
// To add help to your view:
// 1. Add "?" key binding in your view's Update() that navigates to help:
//    case "?":
//        return m, view.NavigateToMsg{ViewName: "help", Payload: GetMyViewHelpContent()}
//
// 2. Add "?" to your view's ShortHelpItems():
//    {Key: "?", Desc: "Help"}
//
// 3. Create a function that returns help categories:
//    func GetMyViewHelpContent() []helpview.HelpCategory {
//        return []helpview.HelpCategory{
//            {Title: "General", Items: []helpview.HelpItem{
//                {Keys: "<key>", Description: "What it does"},
//            }},
//            {Title: "Navigation", Items: []helpview.HelpItem{...}},
//        }
//    }
//
// See views/stacks/update.go for a complete example.

type Model struct {
	Viewable   viewport.Model
	Visible    bool
	content    string
	commands   []CommandInfo
	categories []HelpCategory
	width      int
	height     int
	// app-level "/" filter query
	query string
}

type CommandInfo struct {
	Name        string
	Description string
	Aliases     []string
}

type HelpCategory struct {
	Title string
	Items []HelpItem
}

type HelpItem struct {
	Keys        string
	Description string
}

func New(width, height int, cmds []CommandInfo) *Model {
	cmdW := 16
	descW := 20
	for _, c := range cmds {
		if w := len(c.Name) + 1; w > cmdW { // +1 for ':'
			cmdW = w
		}
		if w := len(c.Description); w > descW {
			descW = w
		}
	}

	const gap = "   "
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	var b strings.Builder
	if len(cmds) > 0 {
		header := fmt.Sprintf("%-*s%s%-*s%s%s", cmdW, "COMMAND", gap, descW, "DESCRIPTION", gap, "ALIASES")
		fmt.Fprintln(&b, headerStyle.Render(header))
	}
	for _, c := range cmds {
		fmt.Fprintf(&b, "%-*s%s%-*s%s%s\n", cmdW, ":"+c.Name, gap, descW, c.Description, gap, strings.Join(c.Aliases, ", "))
	}

	vp := viewport.New(width, height)
	vp.SetContent(b.String())

	return &Model{
		Viewable: vp,
		Visible:  true,
		content:  b.String(),
		commands: cmds,
		width:    width,
		height:   height,
	}
}

func NewDetailed(width, height int, categories []HelpCategory) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	return &Model{
		Viewable:   vp,
		Visible:    true,
		categories: categories,
		width:      width,
		height:     height,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "Close"},
	}
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

// ApplySearchQuery implements view.Filterable — filters commands/keybindings.
func (m *Model) ApplySearchQuery(query string) {
	m.query = query
	if len(m.commands) > 0 {
		m.rebuildCommandContent()
	}
	// Categorized content is rebuilt dynamically in buildCategorizedContent()
}

// ClearSearchQuery implements view.Filterable — clears the filter.
func (m *Model) ClearSearchQuery() {
	m.query = ""
	if len(m.commands) > 0 {
		m.rebuildCommandContent()
	}
}

// rebuildCommandContent re-renders the command list with the current filter.
func (m *Model) rebuildCommandContent() {
	cmds := m.filteredCommands()

	cmdW := 16
	descW := 20
	for _, c := range m.commands {
		if w := len(c.Name) + 1; w > cmdW {
			cmdW = w
		}
		if w := len(c.Description); w > descW {
			descW = w
		}
	}

	const gap = "   "
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")).
		Bold(true)

	var b strings.Builder
	if len(m.commands) > 0 {
		header := fmt.Sprintf("%-*s%s%-*s%s%s", cmdW, "COMMAND", gap, descW, "DESCRIPTION", gap, "ALIASES")
		fmt.Fprintln(&b, headerStyle.Render(header))
	}
	for _, c := range cmds {
		fmt.Fprintf(&b, "%-*s%s%-*s%s%s\n", cmdW, ":"+c.Name, gap, descW, c.Description, gap, strings.Join(c.Aliases, ", "))
	}

	m.content = b.String()
	m.Viewable.SetContent(m.content)
}

// filteredCommands returns commands matching the current query.
func (m *Model) filteredCommands() []CommandInfo {
	if m.query == "" {
		return m.commands
	}
	lower := strings.ToLower(m.query)
	var filtered []CommandInfo
	for _, c := range m.commands {
		if strings.Contains(strings.ToLower(c.Name), lower) ||
			strings.Contains(strings.ToLower(c.Description), lower) {
			filtered = append(filtered, c)
			continue
		}
		for _, alias := range c.Aliases {
			if strings.Contains(strings.ToLower(alias), lower) {
				filtered = append(filtered, c)
				break
			}
		}
	}
	return filtered
}
