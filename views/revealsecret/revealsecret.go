package revealsecretview

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"
	"swarmcli/views/helpbar"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const ViewName = "reveal-secret"

type Model struct {
	viewport   viewport.Model
	secretName string
	content    string
	decoded    bool
	loading    bool
	spinner    int
	width      int
	height     int
}

type revealedMsg struct {
	Content string
	Decoded bool
}

type spinnerTickMsg struct{}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("Loading...")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
		loading:  true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m *Model) Name() string { return ViewName }

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

func (m *Model) SetSecretName(name string) {
	m.secretName = name
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Scroll"},
		{Key: "q/esc", Desc: "Back"},
	}
}

func LoadSecret(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		// Look up secret to get its ID
		secret, err := docker.InspectSecret(ctx, name)
		if err != nil {
			return revealedMsg{Content: fmt.Sprintf("Error inspecting secret: %v", err), Decoded: false}
		}

		// Create temporary service to reveal secret
		serviceName := fmt.Sprintf("swarmcli-reveal-%s-%d", name, time.Now().Unix())
		revealImage := os.Getenv("SWARMCLI_REVEAL_IMAGE")
		if revealImage == "" {
			revealImage = "alpine:latest"
		}
		serviceSpec := docker.CreateSecretRevealServiceWithImage(serviceName, revealImage, secret.Secret.ID, name)

		serviceID, err := docker.CreateService(ctx, serviceSpec)
		if err != nil {
			return revealedMsg{Content: fmt.Sprintf("Error creating reveal service: %v", err), Decoded: false}
		}
		// Always clean up the temporary service.
		defer func() {
			_ = docker.RemoveService(serviceName)
		}()

		// Wait for logs to appear. A fixed sleep is racy (image pull / scheduling).
		var logs string
		deadline := time.Now().Add(20 * time.Second)
		for {
			l, logErr := docker.GetServiceLogs(ctx, serviceID)
			if logErr == nil {
				logs = l
				// If we have any non-newline output, stop polling.
				if strings.TrimRight(logs, "\r\n") != "" {
					break
				}
			}

			if time.Now().After(deadline) {
				if logErr != nil {
					return revealedMsg{Content: fmt.Sprintf("Error getting logs: %v", logErr), Decoded: false}
				}
				diag, diagErr := docker.GetServiceTaskDiagnostics(ctx, serviceID)
				if diagErr != nil {
					return revealedMsg{Content: "(no output from reveal service)\n\nAlso failed to fetch task diagnostics: " + diagErr.Error(), Decoded: false}
				}
				return revealedMsg{Content: "(no output from reveal service)\n\nTask diagnostics:\n" + diag, Decoded: false}
			}

			time.Sleep(300 * time.Millisecond)
		}

		// Preserve raw output for debugging. Only trim line endings from `cat`.
		raw := strings.TrimRight(logs, "\r\n")
		if raw == "" {
			return revealedMsg{Content: "(no output from reveal service)", Decoded: false}
		}

		// Try to detect and decode base64, but keep raw/encoded visible in the UI.
		encodedCandidate := strings.TrimSpace(raw)
		decoded := false
		decodedText := ""

		if encodedCandidate != "" {
			if decodedBytes, err := base64.StdEncoding.DecodeString(encodedCandidate); err == nil && len(decodedBytes) > 0 {
				candidateDecoded := string(decodedBytes)
				if isPrintable(candidateDecoded) {
					decoded = true
					decodedText = candidateDecoded
				}
			}
		}

		content := raw
		if decoded {
			content = fmt.Sprintf("Encoded (base64):\n%s\n\nDecoded:\n%s", encodedCandidate, decodedText)
		}

		return revealedMsg{Content: content, Decoded: decoded}
	}
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.width = msg.Width
		m.height = msg.Height
		return nil

	case tea.KeyMsg:
		// App handles global q/esc; we only handle scrolling keys here.
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd

	case spinnerTickMsg:
		m.spinner++
		if m.loading {
			return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
				return spinnerTickMsg{}
			})
		}
		return nil

	case revealedMsg:
		m.content = msg.Content
		m.decoded = msg.Decoded
		m.loading = false
		m.viewport.SetContent(msg.Content)
		m.viewport.GotoTop()
		return nil
	}

	return nil
}

func (m *Model) View() string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// Title stays stable like inspect; header shows activity/spinner.
	title := fmt.Sprintf("Secret: %s", m.secretName)
	if m.decoded {
		title += " (base64 decoded)"
	}

	header := fmt.Sprintf("Revealing secret: %s", m.secretName)
	if m.loading {
		spinnerChar := ui.SpinnerCharAt(m.spinner)
		header = fmt.Sprintf("Revealing secret: %s %s", m.secretName, spinnerChar)
	}
	headerRendered := ui.FrameHeaderStyle.Render(header)

	// Calculate frame dimensions using full model dimensions like inspect
	frame := ui.ComputeFrameDimensions(
		width,
		m.viewport.Height,
		width,
		m.height,
		headerRendered,
		"",
	)

	// Get viewport content and trim to fit the frame
	viewportContent := ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)

	// Render framed box with no footer (like inspect)
	return ui.RenderFramedBox(title, headerRendered, viewportContent, "", frame.FrameWidth)
}
