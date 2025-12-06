package loadingview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const ViewName = "loading"

// NavigateToContextsMsg is sent when user presses Enter on error
type NavigateToContextsMsg struct{}

type Model struct {
	width, height int
	title         string
	header        string
	message       string
	spinner       spinner.Model
	visible       bool
	isError       bool
}

func New(width, height int, visible bool, payload any) *Model {
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
	isError := strings.HasPrefix(message, "Error")
	return &Model{width: width, height: height, title: title, header: header, message: message, spinner: s, visible: visible, isError: isError}
}

func (m *Model) Visible() bool     { return m.visible }
func (m *Model) SetVisible(v bool) { m.visible = v }
func (m *Model) Init() tea.Cmd     { return m.spinner.Tick }
func (m *Model) Name() string      { return ViewName }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if m.isError && keyMsg.String() == "enter" {
			// Navigate to contexts view
			return func() tea.Msg {
				return NavigateToContextsMsg{}
			}
		}
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return cmd
}

func (m *Model) View() string {
	if !m.visible {
		return ""
	}

	if m.isError {
		// Render error as a styled popup dialog
		return m.renderErrorDialog()
	}

	// Normal loading view
	content := fmt.Sprintf("%s %s", m.spinner.View(), m.message)
	content = strings.TrimSpace(content)
	box := ui.RenderFramedBox(m.title, m.header, content, "", 0) // minimal width
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// renderErrorDialog renders the error dialog with red styling
func (m *Model) renderErrorDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("196")). // Red background for error
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196")) // Red border

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Error "))
	lines = append(lines, itemStyle.Render(""))

	// Wrap error message if too long
	maxWidth := 70
	wrappedLines := wrapText(m.message, maxWidth)
	for _, line := range wrappedLines {
		lines = append(lines, itemStyle.Render(line))
	}

	lines = append(lines, itemStyle.Render(""))
	helpText := fmt.Sprintf("%s %s %s",
		helpStyle.Render("Press"),
		keyStyle.Render("<Enter>"),
		helpStyle.Render("to go to contexts view"))
	lines = append(lines, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	dialog := borderStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// wrapText wraps text to specified width
func wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "Quit"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
