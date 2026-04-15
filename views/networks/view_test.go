// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestView_LoadingState(t *testing.T) {
	m := testModel()
	m.state = stateLoading
	out := m.View()
	require.Contains(t, out, "Docker Networks")
}

func TestView_ReadyState_ShowsNetworks(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("alpha", "beta"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Docker Networks (2)")
	require.Contains(t, out, "alpha")
	require.Contains(t, out, "beta")
}

func TestView_ErrorDialog_ShowsError(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.errorDialogActive = true
	m.err = fmt.Errorf("test error")
	out := m.View()
	require.Contains(t, out, "test error")
}

func TestView_ConfirmDialog(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Delete network 'net1'?"
	out := m.View()
	require.Contains(t, out, "net1")
}

func TestView_CreateDialog_Basic(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "basic"
	out := m.View()
	require.Contains(t, out, "Create Network")
	require.Contains(t, out, "Driver")
}

func TestView_CreateDialog_Review(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "review"
	m.createNameInput.SetValue("mynet")
	out := m.View()
	require.Contains(t, out, "Review")
}

func TestView_CreateDialog_Creating(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "creating"
	out := m.View()
	require.Contains(t, out, "Creating")
}

func TestView_InspectView(t *testing.T) {
	m := testModel()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	m.inspectViewActive = true
	m.inspectContent = `{"name": "mynet"}`
	m.updateInspectViewport()
	out := m.View()
	require.Contains(t, out, "Inspect Network")
}

func TestView_UsedByView(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	m.usedByNetworkName = "my-network"
	m.usedByList.Viewport.Width = 80
	m.usedByList.Viewport.Height = 20
	m.usedByList.Items = []usedByItem{
		{StackName: "stack1", ServiceName: "svc1"},
	}
	m.usedByList.Filtered = m.usedByList.Items
	m.usedByList.RenderItem = func(item usedByItem, _ bool, _ int) string {
		return item.StackName + " " + item.ServiceName
	}
	out := m.View()
	require.Contains(t, out, "my-network")
	require.Contains(t, out, "Used By")
}

func TestView_NameColumnHeader(t *testing.T) {
	m := testModel()
	loadNetworks(m, fakeNetworks("net1"))
	m.setRenderItem()
	m.networksList.Viewport.Width = 80
	m.networksList.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "NAME")
	require.Contains(t, out, "DRIVER")
	require.Contains(t, out, "SCOPE")
}
