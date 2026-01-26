// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"fmt"
	"strings"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
}

type CommandInfo struct {
	Name        string
	Description string
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
	var b strings.Builder
	for _, c := range cmds {
		fmt.Fprintf(&b, ":%-15s %s\n", c.Name, c.Description)
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
