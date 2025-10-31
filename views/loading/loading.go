package loadingview

import (
	"fmt"
	"strings"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"swarmcli/ui"
	"swarmcli/views/view"
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
	frameStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		Padding(1, 2).
		Align(lipgloss.Center)

	content := fmt.Sprintf("%s %s", m.spinner.View(), m.message)
	content = strings.TrimSpace(content)

	frame := frameStyle.Render(content)

	//// Center the frame
	//xPad := max(0, (m.width-lipgloss.Width(frame))/2)
	//yPad := max(0, (m.height-lipgloss.Height(frame))/2)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		frame,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("236")),
	)
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "close"},
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
