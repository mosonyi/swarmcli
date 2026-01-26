// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import "swarmcli/ui"

const Height = 6
const Width = 35

func (m *Model) View() string {
	return ui.StatusStyle.Height(Height).
		Width(Width).
		Render(m.content)
}
