package contexts

import (
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Visible  bool
	viewport viewport.Model
	ready    bool

	contexts             []docker.ContextInfo
	cursor               int
	mu                   sync.Mutex
	loading              bool
	errorMsg             string
	successMsg           string
	switchPending        bool
	confirmDialog        *confirmdialog.Model
	pendingExportContext string
	pendingDeleteContext string
	pendingAction        string // "export" or "delete"
	importInput          textinput.Model
	importInputActive    bool
	fileBrowserActive    bool
	fileBrowserPath      string
	fileBrowserFiles     []string
	fileBrowserCursor    int
}

func New() *Model {
	importInput := textinput.New()
	importInput.Placeholder = "/tmp"
	importInput.Prompt = "Directory: "
	importInput.CharLimit = 512
	importInput.Width = 50

	return &Model{
		Visible:          false,
		contexts:         []docker.ContextInfo{},
		cursor:           0,
		confirmDialog:    confirmdialog.New(0, 0),
		importInput:      importInput,
		fileBrowserPath:  "/tmp",
		fileBrowserFiles: []string{},
	}
}

func (m *Model) SetSize(width, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
	m.confirmDialog.Width = width
	m.confirmDialog.Height = height
	if !m.ready {
		m.ready = true
	}
}

func (m *Model) GetContexts() []docker.ContextInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	contexts := make([]docker.ContextInfo, len(m.contexts))
	copy(contexts, m.contexts)
	return contexts
}

func (m *Model) SetContexts(contexts []docker.ContextInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contexts = contexts
	// Set cursor to current context
	for i, ctx := range contexts {
		if ctx.Current {
			m.cursor = i
			break
		}
	}
}

func (m *Model) GetCursor() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cursor
}

func (m *Model) MoveCursor(delta int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.contexts) {
		m.cursor = len(m.contexts) - 1
	}
}

func (m *Model) GetSelectedContext() (docker.ContextInfo, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cursor >= 0 && m.cursor < len(m.contexts) {
		return m.contexts[m.cursor], true
	}
	return docker.ContextInfo{}, false
}

func (m *Model) SetLoading(loading bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loading = loading
}

func (m *Model) IsLoading() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loading
}

func (m *Model) SetError(err string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorMsg = err
}

func (m *Model) GetError() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.errorMsg
}

func (m *Model) SetSuccess(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successMsg = msg
}

func (m *Model) GetSuccess() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.successMsg
}

func (m *Model) SetSwitchPending(pending bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.switchPending = pending
}

func (m *Model) IsSwitchPending() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.switchPending
}

// HasActiveDialog returns true if any dialog is currently active
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog.Visible || m.importInputActive || m.fileBrowserActive
}

// Init initializes the model (part of View interface)
func (m *Model) Init() tea.Cmd {
	return nil
}

// Name returns the view name (part of View interface)
func (m *Model) Name() string {
	return ViewName
}

// OnEnter is called when the view becomes active
func (m *Model) OnEnter() tea.Cmd {
	m.Visible = true
	return nil
}

// OnExit is called when the view is exited
func (m *Model) OnExit() tea.Cmd {
	m.Visible = false
	return nil
}

// ShortHelpItems returns the help items for the view
func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Switch"},
		{Key: "i", Desc: "Inspect"},
		{Key: "e", Desc: "Export"},
		{Key: "f", Desc: "Import"},
		{Key: "d", Desc: "Delete"},
		{Key: "r", Desc: "Refresh"},
		{Key: "Esc", Desc: "Back"},
	}
}
