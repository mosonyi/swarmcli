package configsview

import (
	"fmt"
	"io"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
)

type configItem struct {
	Name string
	ID   string
}

func (i configItem) FilterValue() string { return i.Name }
func (i configItem) Title() string       { return i.Name }
func (i configItem) Description() string { return fmt.Sprintf("ID: %s", i.ID) }

type itemDelegate struct{}

func (d itemDelegate) Height() int  { return 1 }
func (d itemDelegate) Spacing() int { return 0 }

func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	cfg := item.(configItem)
	selected := index == m.Index()
	if selected {
		fmt.Fprintf(w, "> %s", cfg.Name)
	} else {
		fmt.Fprintf(w, "  %s", cfg.Name)
	}
}

func configItemFromSwarm(c swarm.Config) configItem {
	return configItem{Name: c.Spec.Name, ID: c.ID}
}

func (m *Model) View() string {
	width := 80
	height := 24
	if m.configsList.Viewport.Width > 0 {
		width = m.configsList.Viewport.Width
	}
	if m.configsList.Viewport.Height > 0 {
		height = m.configsList.Viewport.Height
	}

	header := ui.FrameHeaderStyle.Render("DOCKER CONFIGS")

	var content string
	switch m.state {
	case stateLoading:
		content = strings.Repeat("\n", height-1)
	case stateError:
		content = fmt.Sprintf("Error loading configs:\n\n%s\n\nPress q to go back.", m.err)
	case stateReady:
		content = m.configsList.View()
	}

	view := ui.RenderFramedBox("Configs", header, content, "", width)

	if m.confirmDialog.Visible {
		view = ui.OverlayCentered(view, m.confirmDialog.View(), width, height)
	}

	if m.state == stateLoading || m.loadingView.Visible() {
		view = ui.OverlayCentered(view, m.loadingView.View(), width, height)
	}

	return view
}
