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
	message       string
	spinner       spinner.Model
	visible       bool
}

func New(width, height int, message string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.FrameBorderColor)

	return Model{
		width:   width,
		height:  height,
		message: message,
		spinner: s,
		visible: true,
	}
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) Visible() bool {
	return m.visible
}

func (m *Model) SetVisible(v bool) {
	m.visible = v
}

func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	content := fmt.Sprintf("%s %s", m.spinner.View(), m.message)
	content = strings.TrimSpace(content)

	// Center the spinner and message in the available height
	centered := lipgloss.Place(
		m.width,
		m.height-4, // minus borders and header space
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Render the frame using your shared helper
	return ui.RenderFramedBox(
		"Loading",
		"",
		centered,
		m.width,
	)
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "quit"},
	}
}
