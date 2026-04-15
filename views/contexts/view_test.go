// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestView_NotVisible(t *testing.T) {
	m := testModel()
	m.Visible = false
	out := m.View()
	require.Empty(t, out)
}

func TestView_Loading(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetLoading(true)
	m.SetSize(80, 24)
	out := m.View()
	require.Contains(t, out, "Loading contexts")
}

func TestView_WithContexts(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.SetSize(100, 30)
	loadContexts(m, fakeContexts("ctx1", "ctx2"))
	out := m.View()
	require.Contains(t, out, "Docker Contexts")
	require.Contains(t, out, "NAME")
}

func TestView_SwitchPending(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	// Loading + switch pending shows "Switching context..."
	m.SetSwitchPending(true)
	m.SetLoading(true)
	out := m.View()
	// When loading, the header shows status messages
	require.Contains(t, out, "Loading contexts")
}

func TestView_CreateDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	m.createDialogActive = true
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, "Create Docker Context")
}

func TestView_EditDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	m.editDialogActive = true
	m.editContextName = "ctx1"
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, "Edit Context")
	require.Contains(t, out, "ctx1")
}

func TestView_ErrorDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	loadContexts(m, fakeContexts("ctx1"))
	// Set error AFTER loading so ContextsLoadedMsg doesn't clear it
	m.errorDialogActive = true
	m.SetError("Connection refused")
	out := m.View()
	require.Contains(t, out, "Error")
	require.Contains(t, out, "Connection refused")
}

func TestView_ConfirmDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	loadContexts(m, fakeContexts("ctx1"))
	m.confirmDialog = m.confirmDialog.Show("Delete context?")
	out := m.View()
	require.Contains(t, out, "Delete context")
}

func TestView_FileBrowser(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	m.fileBrowserActive = true
	m.fileBrowserPath = "/tmp"
	m.fileBrowserFiles = []string{"..", "/tmp/ctx.tar"}
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, ".tar")
}

func TestView_CertFileBrowser(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(80, 24)
	m.certFileBrowserActive = true
	m.certFileTarget = "ca"
	m.fileBrowserPath = "/home"
	m.fileBrowserFiles = []string{"..", "/home/ca.pem"}
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, "CA Certificate")
}

func TestView_ColumnHeaders(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.SetSize(120, 30)
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "TLS")
	require.Contains(t, out, "DESCRIPTION")
	require.Contains(t, out, "ENDPOINT")
	require.Contains(t, out, "ERROR")
}

func TestView_CreateDialog_TLSEnabled(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	m.SetSize(100, 40)
	m.createDialogActive = true
	m.createTLSEnabled = true
	loadContexts(m, fakeContexts("ctx1"))
	out := m.View()
	require.Contains(t, out, "TLS")
}
