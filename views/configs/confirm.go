package configsview

//func (m Model) handleConfirmDialog(msg tea.Msg) (Model, tea.Cmd, bool) {
//	if !m.confirmDialog.Visible {
//		return m, nil, false
//	}
//
//	switch msg := msg.(type) {
//	case tea.KeyMsg:
//		var cmd tea.Cmd
//		m.confirmDialog, cmd = m.confirmDialog.Update(msg)
//		return m, cmd, true
//	case confirmdialog.ResultMsg:
//		if msg.Confirmed {
//			cfg := m.selectedConfig()
//			switch m.pendingAction {
//			case "rotate":
//				m.pendingAction = ""
//				m.confirmDialog.Visible = false
//				return m, rotateConfigCmd(cfg), true
//			}
//		} else {
//			m.pendingAction = ""
//			m.confirmDialog.Visible = false
//		}
//	}
//	return m, nil, true
//}
