// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"fmt"
	"testing"
	"time"

	"swarmcli/charts"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

// --- State machine tests ---

func TestUpdate_ConfigsLoaded_SetsReady(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("alpha", "beta"))
	require.Equal(t, stateReady, m.state)
	require.Len(t, m.configs, 2)
	require.Len(t, m.configsList.Items, 2)
}

func TestUpdate_ErrorMsg_ShowsError(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.Update(errorMsg(fmt.Errorf("test error")))
	require.True(t, m.errorDialogActive)
}

func TestUpdate_ConfigDeleted_Reloads(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	cmd := m.Update(configDeletedMsg{Name: "c1"})
	require.NotNil(t, cmd)
}

func TestUpdate_ConfigCreated_Reloads(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	cmd := m.Update(configCreatedMsg{Config: swarm.Config{}})
	require.NotNil(t, cmd)
}

func TestUpdate_UsedStatusUpdated(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.Update(usedStatusUpdatedMsg{"id-c1": true})
	require.True(t, m.configsList.Items[0].Used)
	require.True(t, m.configsList.Items[0].UsedKnown)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.configsList.Viewport.Width)
	require.Equal(t, 40, m.configsList.Viewport.Height)
}

func TestUpdate_TickMsg(t *testing.T) {
	m := testModel()
	m.visible = true
	m.state = stateReady
	cmd := m.Update(TickMsg(time.Now()))
	require.NotNil(t, cmd)
}

func TestUpdate_SpinnerTickMsg(t *testing.T) {
	m := testModel()
	old := m.spinner
	m.Update(SpinnerTickMsg(time.Now()))
	require.Equal(t, old+1, m.spinner)
}

func TestUpdate_EditorContentMsg(t *testing.T) {
	m := testModel()
	m.Update(editorContentMsg{Content: "key=value"})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-inline", m.createDialogStep)
	require.Equal(t, "key=value", m.createConfigData)
}

func TestUpdate_FilesLoadedMsg_Success(t *testing.T) {
	m := testModel()
	m.Update(filesLoadedMsg{Path: "/tmp", Files: []string{"..", "foo.txt"}})
	require.True(t, m.fileBrowserActive)
	require.Equal(t, "/tmp", m.fileBrowserPath)
}

func TestUpdate_FilesLoadedMsg_Error(t *testing.T) {
	m := testModel()
	m.Update(filesLoadedMsg{Path: "/nope", Error: fmt.Errorf("no access")})
	require.False(t, m.fileBrowserActive)
	require.True(t, m.createDialogActive)
	require.Contains(t, m.createDialogError, "no access")
}

func TestUpdate_UsedByMsg_Success(t *testing.T) {
	m := testModel()
	m.configsList.Viewport.Width = 80
	m.configsList.Viewport.Height = 20
	m.Update(usedByMsg{ConfigName: "c1", UsedBy: []usedByItem{{StackName: "s1", ServiceName: "svc1"}}})
	require.True(t, m.usedByViewActive)
	require.Equal(t, "c1", m.usedByConfigName)
}

func TestUpdate_UsedByMsg_Error(t *testing.T) {
	m := testModel()
	m.Update(usedByMsg{ConfigName: "c1", Error: fmt.Errorf("fail")})
	require.True(t, m.errorDialogActive)
}

func TestUpdate_ConfigCreateErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(configCreateErrorMsg{err: fmt.Errorf("dup")})
	require.True(t, m.createDialogActive)
	require.Contains(t, m.createDialogError, "dup")
}

func TestUpdate_ConfigRotated_Reloads(t *testing.T) {
	m := testModel()
	cmd := m.Update(configRotatedMsg{})
	require.NotNil(t, cmd)
}

func TestUpdate_EditConfigDoneMsg_Changed(t *testing.T) {
	m := testModel()
	old := docker.ConfigWithDecodedData{Config: swarm.Config{Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "old"}}}}
	new := docker.ConfigWithDecodedData{Config: swarm.Config{Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "new"}}}}
	m.Update(editConfigDoneMsg{Changed: true, OldConfig: old, NewConfig: new})
	require.Equal(t, "rotate", m.pendingAction)
	require.True(t, m.confirmDialog.Visible)
}

func TestUpdate_EditConfigDoneMsg_NoChange(t *testing.T) {
	m := testModel()
	old := docker.ConfigWithDecodedData{Config: swarm.Config{Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "c1"}}}}
	cmd := m.Update(editConfigDoneMsg{Changed: false, OldConfig: old, NewConfig: old})
	require.NotNil(t, cmd) // tea.Printf
}

func TestUpdate_EditConfigErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(editConfigErrorMsg{err: fmt.Errorf("edit failed")})
	require.Equal(t, stateError, m.state)
}

func TestUpdate_EditorContentReadyMsg_Error(t *testing.T) {
	m := testModel()
	m.Update(editorContentReadyMsg{Err: fmt.Errorf("name taken")})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-inline", m.createDialogStep)
	require.Contains(t, m.createDialogError, "name taken")
}

func TestUpdate_FileContentReadyMsg_Error(t *testing.T) {
	m := testModel()
	m.Update(fileContentReadyMsg{FilePath: "/tmp/foo", Err: fmt.Errorf("dup name")})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-file", m.createDialogStep)
	require.Contains(t, m.createDialogError, "dup name")
}

// --- Key routing tests ---

func TestKey_N_OpensCreateDialog(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.Update(key("n"))
	require.True(t, m.createDialogActive)
	require.Equal(t, "source", m.createDialogStep)
}

func TestKey_CtrlD_OpensConfirmDialog(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("myconfig"))
	m.Update(key("ctrl+d"))
	require.True(t, m.confirmDialog.Visible)
	require.Contains(t, m.confirmDialog.Message, "myconfig")
}

func TestKey_I_InspectsConfig(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("myconfig"))
	cmd := m.Update(key("i"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
}

func TestKey_Enter_ViewsRawConfig(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("myconfig"))
	cmd := m.Update(key("enter"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
}

func TestKey_U_OpensUsedByView(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("myconfig"))
	cmd := m.Update(key("u"))
	require.NotNil(t, cmd)
}

func TestKey_Help(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	cmd := m.Update(key("?"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, view.NameHelp, nav.ViewName)
}

func TestKey_ErrorDialog_Dismiss(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.errorDialogActive = true
	m.err = errorMsg(fmt.Errorf("boom"))
	m.Update(key("enter"))
	require.False(t, m.errorDialogActive)
	require.Equal(t, stateReady, m.state)
}

// --- Sort key tests ---

func TestSortKey_N_Name(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("b", "a"))
	// Default is SortByName+asc, N toggles to desc
	m.Update(key("N"))
	require.Equal(t, SortByName, m.sortField)
	require.False(t, m.sortAscending)
}

func TestSortKey_I_ID(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("a", "b"))
	m.Update(key("I"))
	require.Equal(t, SortByID, m.sortField)
}

func TestSortKey_C_Created(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("a", "b"))
	m.Update(key("C"))
	require.Equal(t, SortByCreated, m.sortField)
}

func TestSortKey_D_Updated(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("a", "b"))
	m.Update(key("D"))
	require.Equal(t, SortByUpdated, m.sortField)
}

func TestSortKey_L_Labels(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("a", "b"))
	m.Update(key("L"))
	require.Equal(t, SortByLabels, m.sortField)
}

func TestSortKey_U_Used(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("a", "b"))
	m.Update(key("U"))
	require.Equal(t, SortByUsed, m.sortField)
}

// --- Confirm dialog result tests ---

func TestConfirmResult_Delete(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("target"))
	m.pendingAction = "delete"
	m.configToDelete = &m.configs[0]
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.False(t, m.confirmDialog.Visible)
	require.NotNil(t, cmd)
}

func TestEditBlockedForChartConfig(t *testing.T) {
	m := testModel()
	owned := docker.ConfigWithDecodedData{Config: swarm.Config{
		ID: "id-rel",
		Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{
			Name: "whoami.v1",
			Labels: map[string]string{
				charts.LabelType:    charts.TypeRelease,
				charts.LabelRelease: "whoami",
			},
		}},
	}}
	loadConfigs(m, []docker.ConfigWithDecodedData{owned})

	// Pressing <e> on a chart-owned config opens a dismiss-only info popup
	// explaining why, rather than launching the editor.
	m.Update(key("e"))
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.InfoMode)
	require.Contains(t, m.confirmDialog.Message, "chart release")
	require.Contains(t, m.confirmDialog.Message, "charts upgrade")

	// Dismissing it clears the info mode so a later confirm dialog still works.
	m.Update(confirmdialog.ResultMsg{Confirmed: false})
	require.False(t, m.confirmDialog.InfoMode)
}

func TestDeleteConfirmPrompt(t *testing.T) {
	plain := &docker.ConfigWithDecodedData{Config: swarm.Config{Spec: swarm.ConfigSpec{
		Annotations: swarm.Annotations{Name: "plain"},
	}}}
	require.Equal(t, "Delete config plain?", deleteConfirmPrompt("plain", plain))

	// nil config falls back to the plain prompt.
	require.Equal(t, "Delete config gone?", deleteConfirmPrompt("gone", nil))

	// A config owned by a chart release gets the warning naming the release.
	owned := &docker.ConfigWithDecodedData{Config: swarm.Config{Spec: swarm.ConfigSpec{
		Annotations: swarm.Annotations{Name: "whoami.v1", Labels: map[string]string{
			charts.LabelType:    charts.TypeRelease,
			charts.LabelRelease: "whoami",
		}},
	}}}
	got := deleteConfirmPrompt("whoami.v1", owned)
	require.Contains(t, got, "chart release \"whoami\"")
	require.Contains(t, got, "charts uninstall")
}

func TestConfirmResult_Cancelled(t *testing.T) {
	m := testModel()
	m.pendingAction = "delete"
	m.confirmDialog.Visible = true
	m.Update(confirmdialog.ResultMsg{Confirmed: false})
	require.False(t, m.confirmDialog.Visible)
	require.Equal(t, "", m.pendingAction)
}

// --- Create dialog key tests ---

func TestCreateDialog_Source_Toggle(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createConfigSource = "file"
	m.Update(key("down"))
	require.Equal(t, "inline", m.createConfigSource)
	m.Update(key("up"))
	require.Equal(t, "file", m.createConfigSource)
}

func TestCreateDialog_Source_Enter_File(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createConfigSource = "file"
	m.Update(key("enter"))
	require.Equal(t, "details-file", m.createDialogStep)
}

func TestCreateDialog_Source_Enter_Inline(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createConfigSource = "inline"
	m.Update(key("enter"))
	require.Equal(t, "details-inline", m.createDialogStep)
}

func TestCreateDialog_Source_Esc(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.Update(key("esc"))
	require.False(t, m.createDialogActive)
}

func TestCreateDialog_DetailsFile_Tab(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	m.createInputFocus = 0
	m.Update(key("tab"))
	require.Equal(t, 1, m.createInputFocus)
}

func TestCreateDialog_DetailsFile_Esc(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	m.Update(key("esc"))
	require.False(t, m.createDialogActive)
}

func TestCreateDialog_DetailsFile_EnterEmptyName(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	m.createNameInput.SetValue("")
	m.Update(key("enter"))
	require.Contains(t, m.createDialogError, "name")
}

func TestCreateDialog_DetailsInline_Esc(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.Update(key("esc"))
	require.False(t, m.createDialogActive)
}

func TestCreateDialog_DetailsInline_Tab(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createInputFocus = 0
	m.Update(key("tab"))
	require.Equal(t, 1, m.createInputFocus)
}

// --- Used-by view key tests ---

func TestUsedByView_Esc_Returns(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	m.usedByList.Viewport.Width = 80
	m.usedByList.Viewport.Height = 20
	m.Update(key("esc"))
	require.False(t, m.usedByViewActive)
}

// --- File browser key tests ---

func TestFileBrowser_Esc(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.Update(key("esc"))
	require.False(t, m.fileBrowserActive)
	require.True(t, m.createDialogActive)
}

func TestFileBrowser_UpDown(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserFiles = []string{"..", "file1.txt", "file2.txt"}
	m.fileBrowserCursor = 0
	m.Update(key("down"))
	require.Equal(t, 1, m.fileBrowserCursor)
	m.Update(key("up"))
	require.Equal(t, 0, m.fileBrowserCursor)
}

// --- parseLabels tests ---

func TestParseLabels_Empty(t *testing.T) {
	labels, err := parseLabels("")
	require.NoError(t, err)
	require.Empty(t, labels)
}

func TestParseLabels_Valid(t *testing.T) {
	labels, err := parseLabels("a=b,c=d")
	require.NoError(t, err)
	require.Equal(t, "b", labels["a"])
	require.Equal(t, "d", labels["c"])
}

func TestParseLabels_Invalid(t *testing.T) {
	_, err := parseLabels("bad-format")
	require.Error(t, err)
}

// --- Help content ---

func TestTickMsg_WhenPollingInFlight_SkipsCheck(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.visible = true
	m.polling.Store(true)
	cmd := m.Update(TickMsg(time.Now()))
	require.NotNil(t, cmd)
	// With polling in flight the TickMsg handler should only return tickCmd,
	// not batch(checkConfigsCmd, tickCmd). Executing the returned cmd should
	// produce a TickMsg (from tickCmd), not a PollRetryMsg or configsLoadedMsg.
	msg := runCmd(cmd)
	_, isTickMsg := msg.(TickMsg)
	require.True(t, isTickMsg, "expected TickMsg from tickCmd, got %T", msg)
}

func TestGetConfigsHelpContent(t *testing.T) {
	cats := GetConfigsHelpContent()
	require.True(t, len(cats) >= 3)
	require.Equal(t, "General", cats[0].Title)
	require.Equal(t, "View", cats[1].Title)
	require.Equal(t, "Navigation", cats[2].Title)
}
