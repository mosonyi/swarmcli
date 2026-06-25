// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"testing"

	"swarmcli/settings"
	"swarmcli/views/confirmdialog"

	"github.com/stretchr/testify/require"
)

func TestUpdateNoticeMessage_CE(t *testing.T) {
	origV, origE := version, edition
	defer func() { version, edition = origV, origE }()
	version, edition = "v1.8.0", "ce"

	msg := updateNoticeMessage("v1.9.0")
	require.Contains(t, msg, "v1.9.0")
	require.Contains(t, msg, "v1.8.0")
	require.Contains(t, msg, installDocsURLCommunity)
	require.Contains(t, msg, installDocsURLBusiness) // CE → BE upsell present
	require.Contains(t, msg, "Business Edition")
}

func TestUpdateNoticeMessage_BE(t *testing.T) {
	origV, origE := version, edition
	defer func() { version, edition = origV, origE }()
	version, edition = "v1.8.0", "be"

	msg := updateNoticeMessage("v1.9.0")
	require.Contains(t, msg, installDocsURLBusiness)
	require.Contains(t, msg, "Business Edition")
	require.NotContains(t, msg, "Try Business Edition") // no CE→BE upsell line for BE
}

func TestShowUpdateNotice_SetsInfoDialogWithCheckbox(t *testing.T) {
	m := newTestAppModel(&stubView{name: "test"})
	m.updateDialog = confirmdialog.New(200, 50)

	m.showUpdateNotice("v1.9.0")

	require.True(t, m.updateDialogActive)
	require.True(t, m.updateDialog.Visible)
	require.True(t, m.updateDialog.InfoMode)
	require.Equal(t, updateNoticeCheckbox, m.updateDialog.CheckboxLabel)
	require.False(t, m.updateDialog.CheckboxChecked)
	require.Equal(t, "v1.9.0", m.pendingUpdateVersion)
}

func TestUpdateDialogDismiss_PersistsWhenChecked(t *testing.T) {
	t.Setenv("HOME", t.TempDir()) // redirect settings file location

	m := newTestAppModel(&stubView{name: "test"})
	m.updateDialog = confirmdialog.New(200, 50)
	m.showUpdateNotice("v1.9.0")

	m.Update(confirmdialog.ResultMsg{CheckboxChecked: true})

	require.False(t, m.updateDialogActive)
	require.Equal(t, "v1.9.0", settings.Load().DismissedUpdateVersion)
}

func TestUpdateDialogDismiss_NotPersistedWhenUnchecked(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	m := newTestAppModel(&stubView{name: "test"})
	m.updateDialog = confirmdialog.New(200, 50)
	m.showUpdateNotice("v1.9.0")

	m.Update(confirmdialog.ResultMsg{CheckboxChecked: false})

	require.False(t, m.updateDialogActive)
	require.Equal(t, "", settings.Load().DismissedUpdateVersion)
}
