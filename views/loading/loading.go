// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package loadingview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/ui"
	"github.com/Eldara-Tech/swarmcli/views/helpbar"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const ViewName = "loading"

// SpinnerTickMsg for animating the spinner
type SpinnerTickMsg time.Time

type Model struct {
	width, height int
	title         string
	header        string
	message       string
	spinner       int // frame counter for spinner animation
	visible       bool
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

	return &Model{width: width, height: height, title: title, header: header, message: message, spinner: 0, visible: visible}
}

func (m *Model) Visible() bool        { return m.visible }
func (m *Model) SetVisible(v bool)    { m.visible = v }
func (m *Model) GetSpinnerFrame() int { return m.spinner }
func (m *Model) SetSize(w, h int)     { m.width = w; m.height = h }
func (m *Model) Init() tea.Cmd        { return m.spinnerTickCmd() }
func (m *Model) Name() string         { return ViewName }

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case SpinnerTickMsg:
		m.spinner++
		return m.spinnerTickCmd()
	case tea.WindowSizeMsg:
		// WindowSizeMsg here is already adjusted by the app (height minus systeminfo
		// header and footer), including the columns the app took off for the frame.
		// This view renders its own frame, so it adds them back.
		m.width = msg.Width + ui.FrameChromeColumns
		m.height = msg.Height
		return nil
	}
	return nil
}

func (m *Model) FrameTitle() string  { return m.title }
func (m *Model) FrameHeader() string { return m.header }
func (m *Model) FrameFooter() string { return "" }

func (m *Model) FrameContent() string {
	spinnerChar := ui.SpinnerCharAt(m.spinner)
	return strings.TrimSpace(fmt.Sprintf("%s  %s", spinnerChar, m.message))
}

func (m *Model) View() string {
	if !m.visible {
		return ""
	}

	spinnerChar := ui.SpinnerCharAt(m.spinner)
	content := fmt.Sprintf("%s  %s", spinnerChar, m.message)
	content = strings.TrimSpace(content)
	frameHeight := m.height
	if frameHeight < 0 {
		frameHeight = 0
	}
	frameWidth := m.width
	if frameWidth < 0 {
		frameWidth = 0
	}
	box := ui.RenderFramedBoxHeight(m.title, m.header, content, "", frameWidth, frameHeight)
	return box
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "ctrl+q", Desc: "Quit"},
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
