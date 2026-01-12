package secretsview

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	helpview "swarmcli/views/help"
	loading "swarmcli/views/loading"
	view "swarmcli/views/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// parseLabels parses a comma-separated list of key=value pairs into a map
// Example: "a=b,c=d" -> map[string]string{"a": "b", "c": "d"}
func parseLabels(input string) (map[string]string, error) {
	labels := make(map[string]string)
	if strings.TrimSpace(input) == "" {
		return labels, nil
	}

	pairs := strings.Split(input, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %q (expected key=value)", pair)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("label key cannot be empty in: %q", pair)
		}
		labels[key] = value
	}
	return labels, nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	// Forward spinner ticks to loading view when it's visible
	if m.loadingView.Visible() {
		if _, ok := msg.(loading.SpinnerTickMsg); ok {
			return m.loadingView.Update(msg)
		}
	}

	switch msg := msg.(type) {
	case SpinnerTickMsg:
		// advance spinner and refresh view only if there are unknown UsedKnown items
		m.spinner++
		need := false
		for _, it := range m.secretsList.Items {
			if !it.UsedKnown {
				need = true
				break
			}
		}
		if need {
			m.secretsList.Viewport.SetContent(m.secretsList.View())
		}
		return m.spinnerTickCmd()
	case usedStatusUpdatedMsg:
		l().Infof("SecretsView: Received used status updates for %d secrets", len(msg))
		// Update m.secretsList.Items used flag based on map
		for i := range m.secretsList.Items {
			id := m.secretsList.Items[i].ID
			if used, ok := msg[id]; ok {
				m.secretsList.Items[i].Used = used
				m.secretsList.Items[i].UsedKnown = true
			}
		}

		m.secretsList.Viewport.SetContent(m.secretsList.View())
		return nil

	case tea.WindowSizeMsg:
		m.secretsList.Viewport.Width = msg.Width
		m.secretsList.Viewport.Height = msg.Height
		if m.firstResize {
			m.secretsList.Viewport.YOffset = 0
			m.firstResize = false
		} else if m.secretsList.Cursor == 0 {
			m.secretsList.Viewport.YOffset = 0
		}
		return nil

	case secretsLoadedMsg:
		l().Infof("SecretsView: Received secretsLoadedMsg with %d secrets", len(msg))
		// Update the hash with new data using stable fields only
		type stableSecret struct {
			ID      string
			Version uint64
			Name    string
		}
		stableSecrets := make([]stableSecret, len(msg))
		for i, s := range msg {
			stableSecrets[i] = stableSecret{
				ID:      s.Secret.ID,
				Version: s.Secret.Version.Index,
			}
		}
		var err error
		m.lastSnapshot, err = hash.Compute(stableSecrets)
		if err != nil {
			l().Errorf("SecretsView: Error computing hash: %v", err)
		}

		m.secrets = msg
		items := make([]secretItem, len(msg))
		prevUsed := make(map[string]bool, len(m.secretsList.Items))
		prevKnown := make(map[string]bool, len(m.secretsList.Items))
		for _, it := range m.secretsList.Items {
			prevUsed[it.ID] = it.Used
			prevKnown[it.ID] = it.UsedKnown
		}
		for i, sec := range msg {
			used := false
			known := false
			if val, ok := prevUsed[sec.Secret.ID]; ok {
				used = val
			}
			if k, ok := prevKnown[sec.Secret.ID]; ok {
				known = k
			}
			items[i] = secretItem{
				Name:      sec.Secret.Spec.Name,
				ID:        sec.Secret.ID,
				CreatedAt: sec.Secret.CreatedAt,
				UpdatedAt: sec.Secret.UpdatedAt,
				Labels:    sec.Secret.Spec.Labels,
				Used:      used,
				UsedKnown: known,
			}
		}
		m.secretsList.Items = items
		m.setRenderItem()
		m.secretsList.ApplyFilter()

		m.state = stateReady
		l().Info("SecretsView: Secret list updated (used status pending)")
		return computeSecretUsedCmd(msg)

	case TickMsg:
		l().Infof("SecretsView: Received TickMsg, state=%v, visible=%v", m.state, m.visible)
		if m.visible && m.state == stateReady && !m.confirmDialog.Visible && !m.loadingView.Visible() {
			return tea.Batch(
				CheckSecretsCmd(m.lastSnapshot),
				tickCmd(),
			)
		}
		return tickCmd()

	case secretDeletedMsg:
		l().Infof("Secret deleted successfully: %s", msg.Name)
		return loadSecretsCmd()

	case secretCreatedMsg:
		l().Infof("Secret created successfully: %s", msg.Name)
		m.addSecret(msg.Secret)
		m.createDialogActive = false
		m.createDialogError = ""
		m.createNameInput.Blur()
		m.createFileInput.Blur()
		m.createSecretData = ""
		return tea.Printf("Created secret: %s", msg.Name)

	case fileBrowserMsg:
		m.fileBrowserPath = msg.Path
		m.fileBrowserFiles = msg.Files
		m.fileBrowserCursor = 0
		m.fileBrowserActive = true
		return nil

	case editorContentMsg:
		l().Infof("Editor content received: %d bytes", len(msg.Content))
		m.createSecretData = msg.Content
		// Return to create dialog with inline content
		m.createDialogActive = true
		m.createDialogStep = "details-inline"
		m.createInputFocus = 0
		m.createNameInput.Focus()
		return nil

	case usedByMsg:
		l().Infof("Secret %s is used by %d service(s)", msg.SecretName, len(msg.UsedBy))

		// Initialize usedByList with a new viewport
		w := m.secretsList.Viewport.Width
		if w <= 0 {
			w = m.width
		}
		h := m.secretsList.Viewport.Height
		if h <= 0 {
			if m.height > 0 {
				h = m.height - 2
			}
			if h <= 0 {
				h = 20
			}
		}

		vp := viewport.New(w, h)
		vp.SetContent("")

		m.usedByList = filterlist.FilterableList[usedByItem]{
			Viewport: vp,
			Match: func(item usedByItem, query string) bool {
				return strings.Contains(strings.ToLower(item.StackName), strings.ToLower(query)) ||
					strings.Contains(strings.ToLower(item.ServiceName), strings.ToLower(query))
			},
			RenderItem: func(item usedByItem, selected bool, _ int) string {
				width := vp.Width
				if width <= 0 {
					width = 80
				}
				cols := 2
				starts := make([]int, cols)
				for i := 0; i < cols; i++ {
					starts[i] = (i * width) / cols
				}
				colWidths := make([]int, cols)
				for i := 0; i < cols; i++ {
					if i == cols-1 {
						colWidths[i] = width - starts[i]
					} else {
						colWidths[i] = starts[i+1] - starts[i]
					}
					if colWidths[i] < 1 {
						colWidths[i] = 1
					}
				}

				stackText := item.StackName
				if len(stackText) > colWidths[0] {
					stackText = stackText[:colWidths[0]-1] + "…"
				}
				svcText := item.ServiceName
				if len(svcText) > colWidths[1] {
					svcText = svcText[:colWidths[1]-1] + "…"
				}

				itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
				col0 := itemStyle.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, stackText))
				col1 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[1], svcText))
				line := col0 + col1

				if selected {
					selBg := lipgloss.Color("63")
					selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
					col0 = selStyle.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, stackText))
					col1 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[1], svcText))
					return col0 + col1
				}
				return line
			},
		}

		m.usedByList.Items = msg.UsedBy
		m.usedByList.Viewport.Width = vp.Width
		m.usedByList.Viewport.Height = vp.Height
		m.usedByList.ApplyFilter()

		m.usedBySecretName = msg.SecretName
		m.usedByViewActive = true
		return nil

	case secretRevealedMsg:
		l().Infof("Secret %s revealed: %d bytes (raw)", msg.SecretName, len(msg.Content))
		m.revealDialogActive = true
		m.revealSecretName = msg.SecretName

		// Log first 100 chars of raw content for debugging
		if len(msg.Content) > 0 {
			sample := msg.Content
			if len(sample) > 100 {
				sample = sample[:100] + "..."
			}
			l().Infof("Raw content sample: %q", sample)
		}

		// Try to detect and decode base64
		content := msg.Content
		decoded := false
		trimmed := strings.TrimSpace(content)

		// Only try to decode if content looks like base64:
		// - reasonable length
		// - length is multiple of 4 (or has proper padding)
		// - contains only base64 characters
		if len(trimmed) > 0 {
			if decodedBytes, err := base64.StdEncoding.DecodeString(trimmed); err == nil {
				// Successfully decoded - check if result is printable/reasonable
				decodedStr := string(decodedBytes)
				if len(decodedStr) > 0 {
					content = decodedStr
					decoded = true
					l().Infof("Secret content was base64 encoded, decoded %d -> %d bytes", len(trimmed), len(content))
				}
			} else {
				l().Infof("Content is not base64 encoded (decode error: %v)", err)
			}
		}

		m.revealContent = content
		m.revealDecoded = decoded
		m.revealingInProgress = false
		m.revealViewport.SetContent(content)
		m.revealViewport.GotoTop()
		m.state = stateReady
		return nil

	case errorMsg:
		l().Errorf("Error occurred: %v", msg)
		m.state = stateError
		m.err = msg
		m.errorDialogActive = true
		m.createDialogActive = false
		m.createDialogError = ""
		return nil

	case confirmdialog.ResultMsg:
		l().Debugf("Confirm dialog result: confirmed=%v (pendingAction=%s)", msg.Confirmed, m.pendingAction)
		defer func() {
			m.pendingAction = ""
			m.confirmDialog.Visible = false
			m.secretToDelete = nil
		}()
		if !msg.Confirmed {
			l().Info("Action cancelled by user")
			m.confirmDialog.Visible = false
			return nil
		}

		switch m.pendingAction {
		case "delete":
			if m.secretToDelete == nil {
				l().Warnln("Confirmed delete but secretToDelete is nil")
				return nil
			}
			name := m.secretToDelete.Secret.Spec.Name
			l().Infof("Confirmed deletion for secret %s", name)
			return deleteSecretCmd(name)
		}
		return nil

	case tea.KeyMsg:
		// Handle reveal dialog FIRST before any other key handling
		if m.revealDialogActive {
			return m.handleRevealDialogKey(msg)
		}

		if m.errorDialogActive {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.errorDialogActive = false
				m.err = nil
				m.state = stateReady
				return nil
			}
			return nil
		}

		if m.createDialogActive {
			return m.handleCreateDialogKey(msg)
		}

		if m.usedByViewActive {
			return m.handleUsedByViewKey(msg)
		}

		if m.fileBrowserActive {
			return m.handleFileBrowserKey(msg)
		}

		if m.confirmDialog.Visible {
			l().Debugf("Key input routed to confirm dialog: %q", msg.String())
			return m.confirmDialog.Update(msg)
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.secretsList.Mode == filterlist.ModeSearching {
			m.secretsList.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		if msg.Type == tea.KeyEsc && m.secretsList.Query != "" {
			m.secretsList.Query = ""
			m.secretsList.Mode = filterlist.ModeNormal
			m.secretsList.ApplyFilter()
			m.secretsList.Cursor = 0
			m.secretsList.Viewport.GotoTop()
			return nil
		}

		// Handle specific keys in switch, then navigation keys
		switch msg.String() {
		case "ctrl+d":
			if len(m.secretsList.Filtered) == 0 {
				return nil
			}
			secName := m.selectedSecret()
			sec, _ := m.findSecretByName(secName)
			m.pendingAction = "delete"
			m.secretToDelete = sec
			m.confirmDialog = m.confirmDialog.Show(fmt.Sprintf("Delete secret %s?", secName))
			return nil

		case "u":
			secName := m.selectedSecret()
			if secName == "" {
				l().Warn("UsedBy key pressed but no secret selected")
				return nil
			}
			l().Infof("UsedBy key pressed for secret: %s", secName)
			return getUsedByStacksCmd(secName)

		case "x":
			secName := m.selectedSecret()
			if secName == "" {
				l().Warn("Reveal key pressed but no secret selected")
				return nil
			}
			l().Infof("Reveal key pressed for secret: %s", secName)
			// Push reveal view onto stack (like inspect)
			return pushRevealViewCmd(secName)

		case "left":
			if m.labelsScrollOffset > 0 {
				m.labelsScrollOffset -= 5
				if m.labelsScrollOffset < 0 {
					m.labelsScrollOffset = 0
				}
				m.setRenderItem()
				m.secretsList.Viewport.SetContent(m.secretsList.View())
			}
			return nil

		case "right":
			if m.secretsList.Cursor < len(m.secretsList.Filtered) {
				sec := m.secretsList.Filtered[m.secretsList.Cursor]
				labelsStr := formatLabels(sec.Labels)
				// Allow scrolling if labels are longer than visible width
				if len(labelsStr) > m.labelsScrollOffset+20 {
					m.labelsScrollOffset += 5
					m.setRenderItem()
					m.secretsList.Viewport.SetContent(m.secretsList.View())
				}
			}
			return nil

		case "n":
			l().Info("Create key pressed")
			m.createDialogActive = true
			m.createDialogStep = "source"
			m.createSecretSource = "file" // default
			m.createNameInput.SetValue("")
			m.createSecretData = ""
			m.createDialogError = ""
			return nil

		case "i":
			sec := m.selectedSecret()
			l().Infof("Inspect key pressed for secret: %s", sec)
			return inspectSecretCmd(m.selectedSecret())

		case "?":
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: view.NameHelp,
					Payload:  GetSecretsHelpContent(),
				}
			}

		default:
			// Let FilterableList handle navigation keys (up/down/pgup/pgdown)
			oldCursor := m.secretsList.Cursor
			m.secretsList.HandleKey(msg)
			// Reset scroll offset on cursor movement
			if m.secretsList.Cursor != oldCursor {
				m.labelsScrollOffset = 0
				m.setRenderItem()
				m.secretsList.Viewport.SetContent(m.secretsList.View())
			}
			return nil
		}
	}

	// --- State-based Update ---
	switch m.state {
	case stateReady:
		return nil
	default:
		return nil
	}
}

func (m *Model) setRenderItem() {
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	m.secretsList.RenderItem = func(sec secretItem, selected bool, _ int) string {
		width := m.secretsList.Viewport.Width
		if width <= 0 {
			width = 80
		}

		// Columns: NAME | ID | USED | LABELS | CREATED | UPDATED
		cols := 6
		starts := make([]int, cols)
		for i := 0; i < cols; i++ {
			starts[i] = (i * width) / cols
		}
		colWidths := make([]int, cols)
		for i := 0; i < cols; i++ {
			if i == cols-1 {
				colWidths[i] = width - starts[i]
			} else {
				colWidths[i] = starts[i+1] - starts[i]
			}
			if colWidths[i] < 1 {
				colWidths[i] = 1
			}
		}

		// Ensure CREATED and UPDATED columns have at least 19 chars
		minTime := 19
		cur := colWidths[4] + colWidths[5]
		if cur < 2*minTime {
			deficit := 2*minTime - cur
			for i := 2; i >= 0 && deficit > 0; i-- {
				take := deficit
				if colWidths[i] > take+5 {
					colWidths[i] -= take
					deficit = 0
				} else {
					take = colWidths[i] - 5
					if take > 0 {
						colWidths[i] -= take
						deficit -= take
					}
				}
			}
			if colWidths[4] < minTime {
				colWidths[4] = minTime
			}
			if colWidths[5] < minTime {
				colWidths[5] = minTime
			}
		}

		if colWidths[2] < 1 {
			colWidths[2] = 1
		}

		// Update cached widths for header alignment
		m.colNameWidth = colWidths[0]
		m.colIdWidth = colWidths[1]

		// Prepare cell texts
		nameText := truncateWithEllipsis(sec.Name, colWidths[0]-1)
		idText := truncateWithEllipsis(sec.ID, colWidths[1])
		usedText := " "
		if !sec.UsedKnown {
			usedText = ui.SpinnerCharAt(m.spinner)
		} else if sec.Used {
			usedText = "●"
		}
		// Format timestamps
		createdStr := "N/A"
		if !sec.CreatedAt.IsZero() {
			createdStr = sec.CreatedAt.Format("2006-01-02 15:04:05")
		}
		updatedStr := "N/A"
		if !sec.UpdatedAt.IsZero() {
			updatedStr = sec.UpdatedAt.Format("2006-01-02 15:04:05")
		}
		createdText := truncateWithEllipsis(createdStr, colWidths[3])
		updatedText := truncateWithEllipsis(updatedStr, colWidths[4])
		// Format labels with scroll (sorted and in last column)
		// Reserve 1 char for space before frame end
		maxLabelsWidth := colWidths[5] - 1
		if maxLabelsWidth < 1 {
			maxLabelsWidth = 1
		}
		labelsText := formatLabelsWithScroll(sec.Labels, m.labelsScrollOffset, maxLabelsWidth)

		// Render all columns in one format string (no explicit separators, like nodes view)
		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			return selStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s",
				colWidths[0]-1, nameText,
				colWidths[1], idText,
				colWidths[2], usedText,
				colWidths[3], createdText,
				colWidths[4], updatedText,
				colWidths[5], labelsText,
			))
		}

		return itemStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s",
			colWidths[0]-1, nameText,
			colWidths[1], idText,
			colWidths[2], usedText,
			colWidths[3], createdText,
			colWidths[4], updatedText,
			colWidths[5], labelsText,
		))
	}
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	if maxWidth == 2 {
		return s[:1] + "…"
	}
	return s[:maxWidth-1] + "…"
}

func (m *Model) handleCreateDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch m.createDialogStep {
	case "source":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			return nil
		case "up", "down":
			if m.createSecretSource == "file" {
				m.createSecretSource = "inline"
			} else {
				m.createSecretSource = "file"
			}
			return nil
		case "enter":
			m.createDialogError = ""
			if m.createSecretSource == "file" {
				m.createDialogStep = "details-file"
			} else {
				m.createDialogStep = "details-inline"
			}
			m.createInputFocus = 0
			m.createNameInput.SetValue("")
			m.createFileInput.SetValue("")
			m.createSecretData = ""
			m.createNameInput.Focus()
			m.createFileInput.Blur()
			return nil
		}

	case "details-file":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createFileInput.Blur()
			m.createSecretPath = ""
			m.createInputFocus = 0
			return nil
		case "tab", "shift+tab":
			// Toggle focus between name, file, labels, and encode
			if msg.String() == "tab" {
				m.createInputFocus = (m.createInputFocus + 1) % 4
			} else {
				m.createInputFocus = (m.createInputFocus + 3) % 4
			}
			switch m.createInputFocus {
			case 0:
				m.createNameInput.Focus()
				m.createFileInput.Blur()
				m.createLabelsInput.Blur()
			case 1:
				m.createNameInput.Blur()
				m.createFileInput.Focus()
				m.createLabelsInput.Blur()
			case 2:
				m.createNameInput.Blur()
				m.createFileInput.Blur()
				m.createLabelsInput.Focus()
			default:
				m.createNameInput.Blur()
				m.createFileInput.Blur()
				m.createLabelsInput.Blur()
			}
			return nil
		case " ":
			// Handle space key depending on focused input
			switch m.createInputFocus {
			case 3:
				// Toggle encode when focused on encode toggle
				m.createEncodeSecret = !m.createEncodeSecret
				return nil
			case 0:
				var cmd tea.Cmd
				m.createNameInput, cmd = m.createNameInput.Update(msg)
				return cmd
			case 1:
				var cmd tea.Cmd
				m.createFileInput, cmd = m.createFileInput.Update(msg)
				return cmd
			case 2:
				var cmd tea.Cmd
				m.createLabelsInput, cmd = m.createLabelsInput.Update(msg)
				return cmd
			default:
				return nil
			}
		case "f", "F":
			if m.createInputFocus == 1 {
				m.createDialogActive = false
				m.fileBrowserActive = true
				homeDir, _ := os.UserHomeDir()
				if homeDir == "" {
					homeDir = "/"
				}
				return loadFilesCmd(homeDir)
			}
			// For other focus positions, fall through to default handler
		case "enter":
			if m.createDialogError != "" {
				m.createDialogError = ""
				return nil
			}
			if m.createNameInput.Value() == "" {
				m.createDialogError = "Secret name cannot be empty"
				return nil
			}
			if err := validateSecretName(m.createNameInput.Value()); err != nil {
				m.createDialogError = err.Error()
				return nil
			}
			filePath := m.createFileInput.Value()
			if filePath == "" {
				m.createDialogError = "Please enter or select a file path"
				return nil
			}
			// Parse labels
			labels, err := parseLabels(m.createLabelsInput.Value())
			if err != nil {
				m.createDialogError = fmt.Sprintf("Invalid labels: %v", err)
				return nil
			}
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createFileInput.Blur()
			m.createLabelsInput.Blur()
			return createSecretFromFileCmd(m.createNameInput.Value(), filePath, labels, m.createEncodeSecret)
		default:
			var cmd tea.Cmd
			switch m.createInputFocus {
			case 0:
				m.createNameInput, cmd = m.createNameInput.Update(msg)
			case 1:
				m.createFileInput, cmd = m.createFileInput.Update(msg)
			case 2:
				m.createLabelsInput, cmd = m.createLabelsInput.Update(msg)
			}
			if m.createDialogError != "" {
				m.createDialogError = ""
			}
			return cmd
		}

	case "details-inline":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createSecretData = ""
			m.createInputFocus = 0
			return nil
		case "tab", "shift+tab":
			// Toggle focus between name, content, labels, and encode
			if msg.String() == "tab" {
				m.createInputFocus = (m.createInputFocus + 1) % 4
			} else {
				m.createInputFocus = (m.createInputFocus + 3) % 4
			}
			switch m.createInputFocus {
			case 0:
				m.createNameInput.Focus()
				m.createLabelsInput.Blur()
			case 2:
				m.createNameInput.Blur()
				m.createLabelsInput.Focus()
			default:
				m.createNameInput.Blur()
				m.createLabelsInput.Blur()
			}
			return nil
		case " ":
			// Toggle encode when focused on encode toggle
			if m.createInputFocus == 3 {
				m.createEncodeSecret = !m.createEncodeSecret
				return nil
			}
			// Otherwise pass to focused input
			switch m.createInputFocus {
			case 0:
				var cmd tea.Cmd
				m.createNameInput, cmd = m.createNameInput.Update(msg)
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			case 2:
				var cmd tea.Cmd
				m.createLabelsInput, cmd = m.createLabelsInput.Update(msg)
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			default:
				return nil
			}
		case "e", "E":
			switch m.createInputFocus {
			case 1:
				// Open editor for content - don't require name to be set yet
				m.createDialogActive = false
				m.createNameInput.Blur()
				return openEditorForContentCmd(m.createSecretData)
			default:
				var cmd tea.Cmd
				m.createNameInput, cmd = m.createNameInput.Update(msg)
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			}
		case "enter":
			if m.createDialogError != "" {
				m.createDialogError = ""
				return nil
			}
			if m.createNameInput.Value() == "" {
				m.createDialogError = "Secret name cannot be empty"
				return nil
			}
			if err := validateSecretName(m.createNameInput.Value()); err != nil {
				m.createDialogError = err.Error()
				return nil
			}
			if m.createSecretData == "" {
				m.createDialogError = "Please add content in editor (press Tab then E)"
				return nil
			}
			// Parse labels
			labels, err := parseLabels(m.createLabelsInput.Value())
			if err != nil {
				m.createDialogError = fmt.Sprintf("Invalid labels: %v", err)
				return nil
			}
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createLabelsInput.Blur()
			return createSecretFromContentCmd(m.createNameInput.Value(), []byte(m.createSecretData), labels, m.createEncodeSecret)
		default:
			switch m.createInputFocus {
			case 0:
				var cmd tea.Cmd
				m.createNameInput, cmd = m.createNameInput.Update(msg)
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			case 2:
				var cmd tea.Cmd
				m.createLabelsInput, cmd = m.createLabelsInput.Update(msg)
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			default:
				return nil
			}
		}
	}

	return nil
}

func (m *Model) handleFileBrowserKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.fileBrowserActive = false
		m.createDialogActive = true
		return nil

	case "up":
		if m.fileBrowserCursor > 0 {
			m.fileBrowserCursor--
		}
		return nil

	case "down":
		if m.fileBrowserCursor < len(m.fileBrowserFiles)-1 {
			m.fileBrowserCursor++
		}
		return nil

	case "pgup":
		m.fileBrowserCursor -= 10
		if m.fileBrowserCursor < 0 {
			m.fileBrowserCursor = 0
		}
		return nil

	case "pgdown":
		m.fileBrowserCursor += 10
		if m.fileBrowserCursor >= len(m.fileBrowserFiles) {
			m.fileBrowserCursor = len(m.fileBrowserFiles) - 1
		}
		return nil

	case "enter":
		if len(m.fileBrowserFiles) == 0 {
			return nil
		}

		selected := m.fileBrowserFiles[m.fileBrowserCursor]

		if selected == ".." {
			parentDir := filepath.Dir(m.fileBrowserPath)
			if parentDir == m.fileBrowserPath {
				parentDir = "/"
			}
			return loadFilesCmd(parentDir)
		}

		if strings.HasSuffix(selected, "/") {
			dirPath := strings.TrimSuffix(selected, "/")
			return loadFilesCmd(dirPath)
		}

		// It's a file - set the path and return to dialog
		m.createSecretPath = selected
		m.createFileInput.SetValue(selected)
		m.fileBrowserActive = false
		m.createDialogActive = true
		m.createDialogStep = "details-file"
		m.createInputFocus = 1
		m.createFileInput.Focus()
		return nil
	}
	return nil
}

func (m *Model) handleUsedByViewKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		if m.usedByList.Query != "" {
			m.usedByList.Query = ""
			m.usedByList.Mode = filterlist.ModeNormal
			m.usedByList.ApplyFilter()
			m.usedByList.Cursor = 0
			m.usedByList.Viewport.GotoTop()
			return nil
		}
		m.usedByViewActive = false
		m.usedByList.Items = nil
		m.usedBySecretName = ""
		return nil

	case "enter":
		if len(m.usedByList.Filtered) == 0 {
			return nil
		}
		selectedStack := m.usedByList.Filtered[m.usedByList.Cursor].StackName
		l().Infof("Navigating to services in stack: %s", selectedStack)
		m.usedByViewActive = false
		m.usedByList.Items = nil
		m.usedBySecretName = ""
		return func() tea.Msg {
			payload := map[string]interface{}{"stackName": selectedStack}
			return view.NavigateToMsg{ViewName: view.NameServices, Payload: payload, Replace: false}
		}

	default:
		if m.usedByList.Mode == filterlist.ModeSearching {
			m.usedByList.HandleKey(msg)
		} else {
			m.usedByList.HandleKey(msg)
		}
		return nil
	}
}

func (m *Model) handleRevealDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "q":
		m.revealDialogActive = false
		m.revealSecretName = ""
		m.revealContent = ""
		m.revealDecoded = false
		m.revealingInProgress = false
		m.state = stateReady
		// Don't return anything - the message is consumed
		return tea.Batch()
	case "up", "k":
		m.revealViewport.ScrollUp(1)
		return nil
	case "down", "j":
		m.revealViewport.ScrollDown(1)
		return nil
	case "pgup":
		m.revealViewport.PageUp()
		return nil
	case "pgdown":
		m.revealViewport.PageDown()
		return nil
	case "home", "g":
		m.revealViewport.GotoTop()
		return nil
	case "end", "G":
		m.revealViewport.GotoBottom()
		return nil
	}
	return nil
}

// GetSecretsHelpContent returns categorized help for the secrets view
func GetSecretsHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<n>", Description: "Create new secret"},
				{Keys: "<i>", Description: "Inspect secret (YAML)"},
				{Keys: "<x>", Description: "Reveal secret content"},
				{Keys: "<u>", Description: "Show Used By"},
				{Keys: "<ctrl+d>", Description: "Delete secret"},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name (todo)"},
				{Keys: "<shift+c>", Description: "Order by Created (todo)"},
				{Keys: "<shift+u>", Description: "Order by Updated (todo)"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Navigate"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<esc/q>", Description: "Back to stacks"},
			},
		},
	}
}
