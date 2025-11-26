package servicesview

import (
	"fmt"
	"swarmcli/ui"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	header := ui.FrameHeaderStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		15, "SERVICE",
		10, "STACK",
		10, "REPLICAS",
	))
	content := ui.RenderFramedBox(m.title, header, m.List.View(), "", width)

	if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, m.List.Viewport.Height)
	}
	if m.loading.Visible() {
		content = ui.OverlayCentered(content, m.loading.View(), width, m.List.Viewport.Height)
	}

	return content
}
