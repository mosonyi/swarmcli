package configsview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestView_LoadingState_ShowsLoading(t *testing.T) {
	m := testModel()
	m.state = stateLoading
	out := m.View()
	require.Contains(t, out, "Docker Configs")
}

func TestView_ReadyState_ShowsConfigs(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("alpha", "beta"))
	m.setRenderItem()
	m.configsList.Viewport.Width = 80
	m.configsList.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Docker Configs (2)")
	require.Contains(t, out, "alpha")
	require.Contains(t, out, "beta")
}

func TestView_ErrorDialog_ShowsError(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.setRenderItem()
	m.configsList.Viewport.Width = 80
	m.configsList.Viewport.Height = 20
	m.errorDialogActive = true
	m.err = errorMsg(fmt.Errorf("test error"))
	out := m.View()
	require.Contains(t, out, "test error")
}

func TestView_CreateDialog_Source(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.setRenderItem()
	m.configsList.Viewport.Width = 80
	m.configsList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "source"
	out := m.View()
	require.Contains(t, out, "Create Config")
	require.Contains(t, out, "From file")
}

func TestView_CreateDialog_DetailsFile(t *testing.T) {
	m := testModel()
	loadConfigs(m, fakeConfigs("c1"))
	m.setRenderItem()
	m.configsList.Viewport.Width = 80
	m.configsList.Viewport.Height = 20
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	out := m.View()
	require.Contains(t, out, "Create Config from File")
}

func TestView_UsedByView(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	m.usedByConfigName = "my-config"
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
	require.Contains(t, out, "my-config")
	require.Contains(t, out, "Used By")
}

func TestFormatLabels_Empty(t *testing.T) {
	require.Equal(t, "-", formatLabels(nil))
	require.Equal(t, "-", formatLabels(map[string]string{}))
}

func TestFormatLabels_SortedOutput(t *testing.T) {
	labels := map[string]string{"b": "2", "a": "1"}
	require.Equal(t, "a=1,b=2", formatLabels(labels))
}

func TestFormatLabels_SkipsInternalLabels(t *testing.T) {
	labels := map[string]string{"swarmcli.internal": "val", "env": "prod"}
	require.Equal(t, "env=prod", formatLabels(labels))
}

func TestFormatLabelsWithScroll(t *testing.T) {
	labels := map[string]string{"longkey": "longvalue"}
	full := formatLabels(labels)
	require.Equal(t, "longkey=longvalue", full)

	scrolled := formatLabelsWithScroll(labels, 4, 20)
	require.Equal(t, "key=longvalue", scrolled)
}

func TestFormatLabelsWithScroll_Truncation(t *testing.T) {
	labels := map[string]string{"a": "verylongvalue"}
	truncated := formatLabelsWithScroll(labels, 0, 5)
	require.Contains(t, truncated, ">")
	require.Len(t, truncated, 5)
}
