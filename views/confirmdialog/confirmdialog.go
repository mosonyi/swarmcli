package confirmdialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/ui"
)

type ResultMsg struct {
	Confirmed bool
}

type Model struct {
	Visible bool
	Message string
	Width   int // Parent viewport width
	Height  int // Parent viewport height
}

func New(width, height int) Model {
	return Model{Width: width, Height: height}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			if m.Visible {
				return Model{Visible: false}, func() tea.Msg { return ResultMsg{Confirmed: true} }
			}
		case "n", "N", "esc":
			if m.Visible {
				return Model{Visible: false}, func() tea.Msg { return ResultMsg{Confirmed: false} }
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	// --- Prepare content ---
	lines := []string{
		fmt.Sprintf("⚠️  %s", m.Message),
		"",
		"[y] Yes   [n] No",
	}

	// Compute minimal width
	contentWidth := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > contentWidth {
			contentWidth = w
		}
	}

	hPad, vPad := 2, 1
	boxWidth := contentWidth + hPad*2
	paddedLines := []string{}

	// top padding
	for i := 0; i < vPad; i++ {
		paddedLines = append(paddedLines, strings.Repeat(" ", contentWidth))
	}
	// content lines
	for _, l := range lines {
		left := strings.Repeat(" ", hPad)
		right := strings.Repeat(" ", contentWidth-lipgloss.Width(l))
		paddedLines = append(paddedLines, left+l+right)
	}
	// bottom padding
	for i := 0; i < vPad; i++ {
		paddedLines = append(paddedLines, strings.Repeat(" ", contentWidth))
	}

	boxContent := strings.Join(paddedLines, "\n")

	// --- Render the framed box ---
	overlayBox := ui.RenderFramedBox("Confirm", "", boxContent, boxWidth)

	// --- Center over parent viewport ---
	centered := lipgloss.Place(
		m.Width,
		m.Height,
		lipgloss.Center,
		lipgloss.Center,
		overlayBox,
	)

	return centered
}
