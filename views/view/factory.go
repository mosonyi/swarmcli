// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import tea "github.com/charmbracelet/bubbletea"

type Factory func(width, height int, payload any) (View, tea.Cmd)
