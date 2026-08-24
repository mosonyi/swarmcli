// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// Factory creates a View instance with its initial command.
// The deps parameter provides Docker operation interfaces for testability.
type Factory func(deps docker.Deps, width, height int, payload any) (View, tea.Cmd)
