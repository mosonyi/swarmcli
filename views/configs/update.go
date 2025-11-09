package configsview

import (
	"swarmcli/views/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil

	case configsLoadedMsg:
		items := make([]list.Item, len(msg))
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(cfg.Config)
		}
		m.list.SetItems(items)
		m.state = stateReady
		return m, nil

	case configUpdatedMsg:
		m.list.InsertItem(0, configItemFromSwarm(msg.New.Config))
		return m, tea.Printf("Created new config version: %s", msg.New.Config.Spec.Name)

	case configRotatedMsg:
		return m, tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)

	case editConfigMsg:
		// First, trigger a suspend message so we can safely run the editor next update.
		return m, func() tea.Msg { return tea.Suspend() }

	case tea.SuspendMsg:
		// Now the TUI is safely suspended. Launch the external editor.
		return m, editConfigInEditorCmd(m.selectedConfig())

	case editConfigDoneMsg:
		// Custom message returned after editing finishes successfully.
		return m, tea.Printf("Edited config: %s", msg.Name)

	case editConfigErrorMsg:
		m.state = stateError
		m.err = msg.err
		return m, tea.Printf("Error editing config: %v", msg.err)

	case errorMsg:
		m.state = stateError
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "r":
			//return m, rotateConfigCmd(m.selectedConfig())
		case "e":
			return m, editConfigCmd(m.selectedConfig())
		case "enter":
			return m, inspectConfigCmd(m.selectedConfig())
		}
	}

	switch m.state {
	case stateLoading:
		var cmd tea.Cmd
		m.loadingView, cmd = m.loadingView.Update(msg)
		return m, cmd
	case stateReady:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}
