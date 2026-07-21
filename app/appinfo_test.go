// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

// AppInfoMsg must render the shared modal as a neutral notice (InfoMode), not
// as an error (ErrorMode) — the counterpart to AppErrorMsg.
func TestAppInfoMsg_ShowsNeutralNotice(t *testing.T) {
	m := newTestAppModel(&stubView{name: "test"})

	m.Update(view.AppInfoMsg{Message: "all good", FallbackView: ""})

	require.True(t, m.errorDialog.Visible)
	require.True(t, m.errorDialog.InfoMode)
	require.False(t, m.errorDialog.ErrorMode)
	require.Equal(t, "all good", m.errorDialog.Message)
	require.True(t, m.appErrorDialogActive)
}

// AppErrorMsg keeps its error styling — guards against a regression where the
// two cases share state.
func TestAppErrorMsg_ShowsErrorNotice(t *testing.T) {
	m := newTestAppModel(&stubView{name: "test"})

	m.Update(view.AppErrorMsg{Error: "boom", FallbackView: ""})

	require.True(t, m.errorDialog.Visible)
	require.True(t, m.errorDialog.ErrorMode)
	require.False(t, m.errorDialog.InfoMode)
	require.Equal(t, "boom", m.errorDialog.Message)
}
