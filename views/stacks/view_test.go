package stacksview

import (
	"testing"

	"swarmcli/docker"

	"github.com/stretchr/testify/require"
)

func TestView_NotVisible_Empty(t *testing.T) {
	m := testModel()
	m.Visible = false
	require.Equal(t, "", m.View())
}

func TestView_WithStacks_ShowsTitle(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("alpha", "beta"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Stacks on Node")
	require.Contains(t, out, "Total: 2")
}

func TestView_ShowsStackNames(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("webstack", "apistack"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "apistack")
	require.Contains(t, out, "webstack")
}

func TestView_CreateDialog_Source(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "source"
	out := m.View()
	require.Contains(t, out, "Create Stack")
	require.Contains(t, out, "compose file")
}

func TestView_CreateDialog_DetailsFile(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	out := m.View()
	require.Contains(t, out, "Compose File")
}

func TestView_CreateDialog_DetailsInline(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	out := m.View()
	require.Contains(t, out, "Inline Editor")
}

func TestView_ConfirmDialog(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Remove stack?"
	out := m.View()
	require.Contains(t, out, "Remove stack?")
}

func TestView_ExpandedStackShowsTasks(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	m.expandedStacks["mystack"] = true
	m.stackTasks["mystack"] = []docker.TaskEntry{
		{Name: "mystack_web.1", NodeName: "node1", DesiredState: "running", CurrentState: "running"},
	}
	m.setRenderItem()
	m.List.Viewport.Width = 120
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "mystack_web.1")
	require.Contains(t, out, "node1")
}

func TestView_ErrorColumnHeader(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.stackHasError["s1"] = true
	m.setRenderItem()
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "ERROR: 1")
}
