package loadingview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	"swarmcli/views/helpbar"
	"swarmcli/views/view"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const ViewName = "loading"

type Model struct {
	width, height int
	title         string
	header        string
	message       string
	spinner       spinner.Model
	visible       bool
}

func New(width, height int, visible bool, payload any) Model {
	// Defaults
	title := "Loading"
	header := ""
	message := "Please wait..."

	// --- Auto-detect payload type ---
	switch v := payload.(type) {
	case string:
		message = v
	case map[string]string:
		if t, ok := v["title"]; ok {
			title = t
		}
		if h, ok := v["header"]; ok {
			header = h
		}
		if msg, ok := v["message"]; ok {
			message = msg
		}
	case map[string]interface{}:
		// Support mixed-type maps (consistent with other views)
		if t, ok := v["title"].(string); ok {
			title = t
		}
		if h, ok := v["header"].(string); ok {
			header = h
		}
		if msg, ok := v["message"].(string); ok {
			message = msg
		}
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.FrameBorderColor)

	return Model{
		width:   width,
		height:  height,
		title:   title,
		header:  header,
		message: message,
		spinner: s,
		visible: visible,
	}
}

func (m Model) Name() string { return ViewName }

func (m Model) Visible() bool { return m.visible }

func (m *Model) SetVisible(v bool) { m.visible = v }

func (m Model) Init() tea.Cmd { return m.spinner.Tick }

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	content := fmt.Sprintf("%s %s", m.spinner.View(), m.message)
	content = strings.TrimSpace(content)

	centered := lipgloss.Place(
		m.width,
		m.height-4, // leave room for frame
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	return ui.RenderFramedBox(
		m.title,
		m.header,
		centered,
		m.width,
	)
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "quit"},
	}
}
