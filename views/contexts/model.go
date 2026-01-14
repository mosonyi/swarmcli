package contexts

import (
	"strings"
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	filterlist "swarmcli/ui/components/filterable/list"
)

type SortField int

const (
	SortByName SortField = iota
	SortByStatus
	SortByDescription
	SortByEndpoint
)

type Model struct {
	Visible  bool
	viewport viewport.Model
	ready    bool

	List filterlist.FilterableList[docker.ContextInfo]

	sortField     SortField
	sortAscending bool // true for ascending, false for descending

	contexts              []docker.ContextInfo
	cursor                int
	mu                    sync.Mutex
	loading               bool
	errorMsg              string
	successMsg            string
	switchPending         bool
	confirmDialog         *confirmdialog.Model
	pendingExportContext  string
	pendingDeleteContext  string
	pendingAction         string // "export" or "delete"
	importInput           textinput.Model
	importInputActive     bool
	fileBrowserActive     bool
	fileBrowserPath       string
	fileBrowserFiles      []string
	fileBrowserCursor     int
	errorDialogActive     bool
	createDialogActive    bool
	createNameInput       textinput.Model
	createDescInput       textinput.Model
	createHostInput       textinput.Model
	createInputFocus      int // 0 = name, 1 = description, 2 = host, 3 = tls toggle, 4 = ca, 5 = cert, 6 = key
	createTLSEnabled      bool
	createCAInput         textinput.Model
	createCertInput       textinput.Model
	createKeyInput        textinput.Model
	certFileBrowserActive bool   // true when browsing for cert files (different from import file browser)
	certFileTarget        string // "ca", "cert", or "key" - which field is being browsed
	lastCertBrowserPath   string // Remember last directory used in cert file browser
	editDialogActive      bool
	editContextName       string // Name of context being edited (immutable)
	editDescInput         textinput.Model
}

func New() *Model {
	importInput := textinput.New()
	importInput.Placeholder = "/tmp"
	importInput.Prompt = "Directory: "
	importInput.CharLimit = 512
	importInput.Width = 50

	createNameInput := textinput.New()
	createNameInput.Placeholder = "my-context"
	createNameInput.Prompt = "Name: "
	createNameInput.CharLimit = 100
	createNameInput.Width = 50

	createDescInput := textinput.New()
	createDescInput.Placeholder = "Description (optional)"
	createDescInput.Prompt = "Desc: "
	createDescInput.CharLimit = 200
	createDescInput.Width = 50

	createHostInput := textinput.New()
	createHostInput.Placeholder = "tcp://host:2376"
	createHostInput.Prompt = "Host: "
	createHostInput.CharLimit = 256
	createHostInput.Width = 50

	createCAInput := textinput.New()
	createCAInput.Placeholder = "/path/to/ca.pem"
	createCAInput.Prompt = "CA:   "
	createCAInput.CharLimit = 512
	createCAInput.Width = 50

	createCertInput := textinput.New()
	createCertInput.Placeholder = "/path/to/cert.pem"
	createCertInput.Prompt = "Cert: "
	createCertInput.CharLimit = 512
	createCertInput.Width = 50

	createKeyInput := textinput.New()
	createKeyInput.Placeholder = "/path/to/key.pem"
	createKeyInput.Prompt = "Key:  "
	createKeyInput.CharLimit = 512
	createKeyInput.Width = 50

	editDescInput := textinput.New()
	editDescInput.Placeholder = "Description (optional)"
	editDescInput.Prompt = "Desc: "
	editDescInput.CharLimit = 200
	editDescInput.Width = 50

	// Initialize an internal viewport for the filterable list
	vp := viewport.New(80, 20)
	vp.SetContent("")

	list := filterlist.FilterableList[docker.ContextInfo]{
		Viewport: vp,
		Match: func(item docker.ContextInfo, query string) bool {
			return strings.Contains(strings.ToLower(item.Name), strings.ToLower(query))
		},
	}

	return &Model{
		Visible:          false,
		contexts:         []docker.ContextInfo{},
		cursor:           0,
		confirmDialog:    confirmdialog.New(0, 0),
		importInput:      importInput,
		fileBrowserPath:  "/tmp",
		fileBrowserFiles: []string{},
		createNameInput:  createNameInput,
		createDescInput:  createDescInput,
		createHostInput:  createHostInput,
		createCAInput:    createCAInput,
		createCertInput:  createCertInput,
		createKeyInput:   createKeyInput,
		editDescInput:    editDescInput,
		List:             list,
		sortField:        SortByName,
		sortAscending:    true,
	}
}

func (m *Model) SetSize(width, height int) {
	m.viewport.Width = width
	m.viewport.Height = height
	m.confirmDialog.Width = width
	m.confirmDialog.Height = height
	// Keep the internal list viewport in sync so it doesn't stay at its
	// initial 80x20 size when the view receives data.
	if width > 0 {
		m.List.Viewport.Width = width
	}
	if height > 0 {
		// Reserve 2 lines for stackbar/bottom status like other views
		h := height - 2
		if h <= 0 {
			h = 20
		}
		m.List.Viewport.Height = h
	}
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

	// Preserve cursor position if possible
	oldCursor := m.cursor
	m.contexts = contexts

	// If this is the first load (cursor at 0 and contexts were empty), set to current
	if oldCursor == 0 && len(contexts) > 0 {
		for i, ctx := range contexts {
			if ctx.Current {
				m.cursor = i
				break
			}
		}
	}

	// Otherwise keep cursor position, but validate bounds
	if m.cursor >= len(m.contexts) {
		m.cursor = len(m.contexts) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Update the FilterableList backing items and apply filter
	m.List.Items = m.contexts
	// Keep FilterableList cursor in sync with our cursor so keyboard navigation
	// that manipulates m.cursor affects the list selection and visible offset.
	m.List.Cursor = m.cursor
	// Ensure the list viewport matches the current view size so the
	// content fills the frame immediately when contexts arrive.
	if m.viewport.Width > 0 {
		m.List.Viewport.Width = m.viewport.Width
	}
	if m.viewport.Height > 0 {
		h := m.viewport.Height
		if h <= 0 {
			h = 20
		}
		m.List.Viewport.Height = h
	}
	m.List.ApplyFilter()
	m.applySorting()
	// Update the internal viewport content immediately so parent view
	// that uses the viewport's content (e.g., during initial render)
	// doesn't keep showing the loading placeholder.
	m.List.Viewport.SetContent(m.List.View())
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
	// Mirror to internal FilterableList cursor and ensure visible
	if m.List.Cursor != m.cursor {
		m.List.Cursor = m.cursor
		// ApplyFilter will keep the cursor in-bounds and update viewport offset
		m.List.ApplyFilter()
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
	// Write debug state when loading changes
	go m.debugWriteLoadingState()
}

// debug write of loading state for diagnostics
func (m *Model) debugWriteLoadingState() {
	// Only log this information at debug level using the standard logger.
	// The logger will no-op if not running in debug mode.
	swarmlog.L().Debugf("[contexts] loading=%v count=%d", m.loading, len(m.contexts))
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
	return m.confirmDialog.Visible || m.importInputActive || m.fileBrowserActive || m.errorDialogActive || m.createDialogActive || m.certFileBrowserActive || m.editDialogActive
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
	// If we have no contexts loaded, trigger a load on enter so the
	// view shows progress and fills itself. Also allow explicit reloads
	// from other code paths by calling SetLoading(true) before navigating.
	if len(m.contexts) == 0 {
		m.SetLoading(true)
		return func() tea.Msg { return LoadContextsCmd() }
	}
	return nil
}

// OnExit is called when the view is exited
func (m *Model) OnExit() tea.Cmd {
	m.Visible = false
	return nil
}

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.List.Query != ""
}

// IsSearching reports whether the list is currently in search mode.
func (m *Model) IsSearching() bool {
	return m.List.Mode == filterlist.ModeSearching
}

// updateCreateFocus updates focus state for create dialog inputs
func (m *Model) updateCreateFocus() {
	m.createNameInput.Blur()
	m.createDescInput.Blur()
	m.createHostInput.Blur()
	m.createCAInput.Blur()
	m.createCertInput.Blur()
	m.createKeyInput.Blur()

	switch m.createInputFocus {
	case 0:
		m.createNameInput.Focus()
	case 1:
		m.createDescInput.Focus()
	case 2:
		m.createHostInput.Focus()
	case 4:
		m.createCAInput.Focus()
	case 5:
		m.createCertInput.Focus()
	case 6:
		m.createKeyInput.Focus()
		// case 3 is the TLS checkbox, no focus needed
	}
} // ShortHelpItems returns the help items for the view
func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "Enter", Desc: "Switch"},
		{Key: "i", Desc: "Inspect"},
		{Key: "e", Desc: "Edit"},
		{Key: "x", Desc: "Export"},
		{Key: "m", Desc: "Import"},
		{Key: "c", Desc: "Create"},
		{Key: "d", Desc: "Delete"},
		{Key: "?", Desc: "Help"},
		{Key: "Esc", Desc: "Back"},
	}
}
