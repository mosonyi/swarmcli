package stacksview

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// --- State machine tests ---

func TestUpdate_Msg_SetsVisibleAndStacks(t *testing.T) {
	m := testModel()
	m.Update(Msg{NodeID: "n1", Stacks: fakeStacks("alpha", "beta")})
	require.True(t, m.Visible)
	require.Len(t, m.List.Items, 2)
	require.Len(t, m.List.Filtered, 2)
}

func TestUpdate_Msg_LaunchesTaskFetches(t *testing.T) {
	fetched := map[string]bool{}
	taskMock := &mockTaskOps{
		getTasksForStackFn: func(name string) ([]docker.TaskEntry, error) {
			fetched[name] = true
			return nil, nil
		},
		getTasksForServiceFn: func(_ string) ([]docker.TaskEntry, error) { return nil, nil },
	}
	m := testModel(func(m *Model) { m.deps.Tasks = taskMock })
	cmd := m.Update(Msg{NodeID: "n1", Stacks: fakeStacks("s1", "s2")})
	require.NotNil(t, cmd)
	// The batched commands include tickCmd + task fetch cmds.
	// We only need to verify the mock was called; the tick produces a time.Timer
	// that we skip. Walk the batch and run only non-tick commands.
	batchMsg := cmd()
	if batch, ok := batchMsg.(tea.BatchMsg); ok {
		for _, c := range batch {
			if c == nil {
				continue
			}
			// Run the cmd; tick cmds block on time.After, so use a goroutine + timeout
			done := make(chan struct{})
			go func() {
				defer close(done)
				c()
			}()
			select {
			case <-done:
			case <-time.After(50 * time.Millisecond):
				// tick cmd — skip it
			}
		}
	}
	require.True(t, fetched["s1"])
	require.True(t, fetched["s2"])
}

func TestUpdate_TickMsg_WhenVisible(t *testing.T) {
	m := testModel()
	m.Visible = true
	cmd := m.Update(TickMsg(time.Now()))
	require.NotNil(t, cmd)
}

func TestUpdate_TickMsg_WhenNotVisible(t *testing.T) {
	m := testModel()
	m.Visible = false
	cmd := m.Update(TickMsg(time.Now()))
	// Should still return tick cmd for polling
	require.NotNil(t, cmd)
}

func TestUpdate_RefreshErrorMsg_SetsContent(t *testing.T) {
	m := testModel()
	m.Update(RefreshErrorMsg{Err: fmt.Errorf("timeout")})
	require.True(t, m.Visible)
}

func TestUpdate_StackTasksLoadedMsg_CachesTasks(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	tasks := []docker.TaskEntry{
		{Name: "mystack_web.1", NodeName: "node1"},
	}
	m.Update(StackTasksLoadedMsg{StackName: "mystack", Tasks: tasks})
	require.Len(t, m.stackTasks["mystack"], 1)
}

func TestUpdate_StackTasksLoadedMsg_ErrorCachesEmpty(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	m.Update(StackTasksLoadedMsg{StackName: "mystack", Error: fmt.Errorf("fail")})
	require.Empty(t, m.stackTasks["mystack"])
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.width)
	require.Equal(t, 40, m.height)
	require.True(t, m.ready)
}

func TestUpdate_RemoveErrorMsg_ShowsError(t *testing.T) {
	m := testModel()
	m.Update(RemoveErrorMsg{StackName: "bad", Error: fmt.Errorf("permission denied")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "bad")
}

func TestUpdate_StackUpdateErrorMsg_ShowsError(t *testing.T) {
	m := testModel()
	m.Update(stackUpdateErrorMsg{StackName: "mystack", Err: fmt.Errorf("invalid yaml")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "mystack")
}

func TestUpdate_StackCreateErrorMsg_ReturnsToDialog(t *testing.T) {
	m := testModel()
	m.createFileInput.SetValue("somefile.yml")
	m.Update(stackCreateErrorMsg{Err: fmt.Errorf("deployment failed")})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-file", m.createDialogStep)
	require.Contains(t, m.createDialogError, "deployment failed")
}

func TestUpdate_StackCreateErrorMsg_InlineMode(t *testing.T) {
	m := testModel()
	// No file input value → returns to inline step
	m.Update(stackCreateErrorMsg{Err: fmt.Errorf("bad yaml")})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-inline", m.createDialogStep)
}

func TestUpdate_EditorContentMsg_CreateMode(t *testing.T) {
	m := testModel()
	m.editStackName = "" // create mode
	m.Update(editorContentMsg{Content: "version: '3'\nservices:\n  web:\n    image: nginx"})
	require.True(t, m.createDialogActive)
	require.Equal(t, "details-inline", m.createDialogStep)
	require.Contains(t, m.createDialogContent, "nginx")
}

func TestUpdate_EditorContentMsg_EditMode(t *testing.T) {
	deployed := ""
	stackMock := noopStackOps()
	stackMock.deployStackFn = func(name string, content string) error {
		deployed = name
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	m.editStackName = "mystack"
	cmd := m.Update(editorContentMsg{Content: "version: '3'", OriginalContent: "version: '2'"})
	require.Equal(t, "", m.editStackName) // cleared
	require.NotNil(t, cmd)
	runCmd(cmd)
	require.Equal(t, "mystack", deployed)
}

func TestUpdate_EditorContentMsg_EditMode_NoChange(t *testing.T) {
	deployed := false
	stackMock := noopStackOps()
	stackMock.deployStackFn = func(name string, content string) error {
		deployed = true
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	m.editStackName = "mystack"
	yaml := "version: '3'\nservices:\n  web:\n    image: nginx"
	cmd := m.Update(editorContentMsg{Content: yaml, OriginalContent: yaml})
	require.Equal(t, "", m.editStackName) // cleared
	require.Nil(t, cmd)                   // no redeploy
	require.False(t, deployed)
}

func TestUpdate_FilesLoadedMsg_Success(t *testing.T) {
	m := testModel()
	m.Update(filesLoadedMsg{Path: "/tmp", Files: []string{"..", "foo.yml"}})
	require.True(t, m.fileBrowserActive)
	require.Equal(t, "/tmp", m.fileBrowserPath)
	require.Len(t, m.fileBrowserFiles, 2)
}

func TestUpdate_FilesLoadedMsg_Error(t *testing.T) {
	m := testModel()
	m.Update(filesLoadedMsg{Path: "/nope", Error: fmt.Errorf("no access")})
	require.False(t, m.fileBrowserActive)
	require.True(t, m.createDialogActive)
	require.Contains(t, m.createDialogError, "no access")
}

// --- Key routing tests ---

func TestKey_N_OpensCreateDialog(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.Update(key("n"))
	require.True(t, m.createDialogActive)
	require.Equal(t, "source", m.createDialogStep)
}

func TestKey_C_OpensCreateDialog(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.Update(key("c"))
	require.True(t, m.createDialogActive)
	require.Equal(t, "source", m.createDialogStep)
}

func TestKey_CtrlD_OpensConfirmDialog(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	m.Update(key("ctrl+d"))
	require.True(t, m.confirmDialog.Visible)
	require.Equal(t, "remove", m.pendingAction)
	require.Contains(t, m.confirmDialog.Message, "mystack")
}

func TestKey_P_TogglesExpand(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.Update(key("p"))
	require.True(t, m.expandedStacks["s1"])
	// Toggle off
	m.Update(key("p"))
	require.False(t, m.expandedStacks["s1"])
}

func TestKey_Enter_NavigatesToServices(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	cmd := m.Update(key("enter"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "services", nav.ViewName)
}

func TestKey_I_Inspect(t *testing.T) {
	inspected := ""
	stackMock := noopStackOps()
	stackMock.inspectStackFn = func(name string) (string, error) {
		inspected = name
		return "yaml-content", nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("mystack"))
	cmd := m.Update(key("i"))
	msg := runCmd(cmd)
	require.Equal(t, "mystack", inspected)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
}

func TestKey_I_InspectError(t *testing.T) {
	stackMock := noopStackOps()
	stackMock.inspectStackFn = func(_ string) (string, error) {
		return "", fmt.Errorf("not found")
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("mystack"))
	cmd := m.Update(key("i"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	payload := nav.Payload.(map[string]any)
	require.Contains(t, payload["title"].(string), "inspect failed")
}

func TestKey_Help(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	cmd := m.Update(key("?"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, view.NameHelp, nav.ViewName)
}

func TestKey_Esc_ClearsFilter(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("alpha", "beta"))
	m.List.Query = "alpha"
	m.Update(key("esc"))
	require.Equal(t, "", m.List.Query)
}

// --- Sort key tests ---

func TestSortKey_S_Name(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("beta", "alpha"))
	// Default is SortByName+asc; first S toggles to desc
	m.Update(key("S"))
	require.Equal(t, SortByName, m.sortField)
	require.True(t, m.userSetSort)
	require.False(t, m.sortAscending)
	// Press again to toggle back to ascending
	m.Update(key("S"))
	require.True(t, m.sortAscending)
}

func TestSortKey_E_Services(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("a", "b"))
	m.Update(key("E"))
	require.Equal(t, SortByServices, m.sortField)
}

func TestSortKey_T_Tasks(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("a", "b"))
	m.Update(key("T"))
	require.Equal(t, SortByTasks, m.sortField)
}

func TestSortKey_R_Error(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("a", "b"))
	m.Update(key("R"))
	require.Equal(t, SortByError, m.sortField)
}

// --- Confirm dialog result tests ---

func TestConfirmResult_Remove(t *testing.T) {
	removed := ""
	stackMock := noopStackOps()
	stackMock.removeStackFn = func(_ context.Context, name string) error {
		removed = name
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("target"))
	m.pendingAction = "remove"
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.False(t, m.confirmDialog.Visible)
	runCmd(cmd)
	require.Equal(t, "target", removed)
}

func TestConfirmResult_RemoveWithNetworks(t *testing.T) {
	removedNetworks := ""
	stackMock := noopStackOps()
	stackMock.removeStackFn = func(_ context.Context, _ string) error { return nil }
	stackMock.removeStackNetworksFn = func(_ context.Context, name string) error {
		removedNetworks = name
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("target"))
	m.pendingAction = "remove"
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true, CheckboxChecked: true})
	runCmd(cmd)
	require.Equal(t, "target", removedNetworks)
}

func TestConfirmResult_Cancelled(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.pendingAction = "remove"
	m.confirmDialog.Visible = true
	m.Update(confirmdialog.ResultMsg{Confirmed: false})
	require.False(t, m.confirmDialog.Visible)
	require.Equal(t, "", m.pendingAction)
}

// --- Create dialog tests ---

func TestCreateDialog_Source_Toggle(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createStackSource = "file"
	m.Update(key("down"))
	require.Equal(t, "inline", m.createStackSource)
	m.Update(key("up"))
	require.Equal(t, "file", m.createStackSource)
}

func TestCreateDialog_Source_Enter_File(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createStackSource = "file"
	m.Update(key("enter"))
	require.Equal(t, "details-file", m.createDialogStep)
}

func TestCreateDialog_Source_Enter_Inline(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "source"
	m.createStackSource = "inline"
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
	m.Update(key("tab"))
	require.Equal(t, 0, m.createInputFocus)
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
	require.Contains(t, m.createDialogError, "Stack name cannot be empty")
}

func TestCreateDialog_DetailsFile_EnterEmptyPath(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-file"
	m.createNameInput.SetValue("mystack")
	m.createFileInput.SetValue("")
	m.Update(key("enter"))
	require.Contains(t, m.createDialogError, "file path")
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

func TestCreateDialog_DetailsInline_EnterEmptyName(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createNameInput.SetValue("")
	m.Update(key("enter"))
	require.Contains(t, m.createDialogError, "Stack name cannot be empty")
}

func TestCreateDialog_DetailsInline_EnterNoContent(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createNameInput.SetValue("mystack")
	m.createDialogContent = ""
	m.Update(key("enter"))
	require.Contains(t, m.createDialogError, "YAML content")
}

func TestCreateDialog_DetailsInline_EnterValidation(t *testing.T) {
	stackMock := noopStackOps()
	stackMock.validateStackYAMLFn = func(_ string) error {
		return fmt.Errorf("invalid services")
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createNameInput.SetValue("mystack")
	m.createDialogContent = "bad yaml"
	m.Update(key("enter"))
	require.Contains(t, m.createDialogError, "invalid services")
}

func TestCreateDialog_DetailsInline_EnterDeploys(t *testing.T) {
	deployed := ""
	stackMock := noopStackOps()
	stackMock.validateStackYAMLFn = func(_ string) error { return nil }
	stackMock.deployStackFn = func(name string, _ string) error {
		deployed = name
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createNameInput.SetValue("mystack")
	m.createDialogContent = "version: '3'\nservices:\n  web:\n    image: nginx"
	cmd := m.Update(key("enter"))
	require.False(t, m.createDialogActive)
	runCmd(cmd)
	require.Equal(t, "mystack", deployed)
}

func TestCreateDialog_EnterWithError_ClearsError(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	m.createDialogStep = "details-inline"
	m.createDialogError = "previous error"
	m.Update(key("enter"))
	require.Equal(t, "", m.createDialogError)
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
	m.fileBrowserFiles = []string{"..", "file1.yml", "file2.yml"}
	m.fileBrowserCursor = 0
	m.Update(key("down"))
	require.Equal(t, 1, m.fileBrowserCursor)
	m.Update(key("up"))
	require.Equal(t, 0, m.fileBrowserCursor)
	// Don't go below 0
	m.Update(key("up"))
	require.Equal(t, 0, m.fileBrowserCursor)
}

func TestFileBrowser_PgUpDown(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	files := make([]string, 20)
	for i := range files {
		files[i] = fmt.Sprintf("file%d.yml", i)
	}
	m.fileBrowserFiles = files
	m.fileBrowserCursor = 0
	m.Update(key("pgdown"))
	require.Equal(t, 10, m.fileBrowserCursor)
	m.Update(key("pgup"))
	require.Equal(t, 0, m.fileBrowserCursor)
}

func TestFileBrowser_Enter_ParentDir(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserFiles = []string{".."}
	m.fileBrowserPath = "/home/user"
	m.fileBrowserCursor = 0
	cmd := m.Update(key("enter"))
	// Should return a loadFilesCmd for parent dir
	require.NotNil(t, cmd)
}

func TestFileBrowser_Enter_Directory(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserFiles = []string{"/tmp/subdir/"}
	m.fileBrowserCursor = 0
	cmd := m.Update(key("enter"))
	require.NotNil(t, cmd)
}

func TestRemoveStack_Timeout_ReturnsRemoveError(t *testing.T) {
	stackMock := noopStackOps()
	stackMock.removeStackFn = func(_ context.Context, _ string) error {
		return context.DeadlineExceeded
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("target"))
	m.pendingAction = "remove"
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
	msg := runCmd(cmd)
	_, ok := msg.(RemoveErrorMsg)
	require.True(t, ok, "expected RemoveErrorMsg, got %T", msg)
}

// --- Task navigation tests ---

func TestTaskNavigation_DownIntoTasks(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.expandedStacks["s1"] = true
	m.stackTasks["s1"] = []docker.TaskEntry{
		{Name: "s1_web.1"}, {Name: "s1_web.2"},
	}
	m.selectedTaskIndex = -1
	m.Update(key("down"))
	require.Equal(t, 0, m.selectedTaskIndex)
	m.Update(key("down"))
	require.Equal(t, 1, m.selectedTaskIndex)
}

func TestTaskNavigation_UpFromTasks(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.expandedStacks["s1"] = true
	m.stackTasks["s1"] = []docker.TaskEntry{{Name: "s1_web.1"}}
	m.selectedTaskIndex = 0
	m.Update(key("up"))
	require.Equal(t, -1, m.selectedTaskIndex)
}

// --- Horizontal scroll tests ---

func TestHorizontalScroll_Right(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.stackTasks["s1"] = []docker.TaskEntry{
		{DesiredState: "running", Error: "something went wrong"},
	}
	m.Update(key("right"))
	require.Equal(t, 5, m.errorScrollOffset)
}

func TestHorizontalScroll_Left(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("s1"))
	m.errorScrollOffset = 10
	m.Update(key("left"))
	require.Equal(t, 5, m.errorScrollOffset)
}

// --- Save dialog tests ---

func TestKey_S_OpensSaveDialog(t *testing.T) {
	m := testModel()
	loadStacks(m, fakeStacks("mystack"))
	m.Update(key("s"))
	require.True(t, m.saveDialogActive)
	require.Equal(t, "mystack", m.saveStackName)
	require.Equal(t, "mystack.yml", m.saveFileInput.Value())
}

func TestKey_S_NoStacks_NoDialog(t *testing.T) {
	m := testModel()
	m.Update(key("s"))
	require.False(t, m.saveDialogActive)
}

func TestSaveDialog_Esc_Closes(t *testing.T) {
	m := testModel()
	m.saveDialogActive = true
	m.saveStackName = "mystack"
	m.saveDialogError = "some error"
	m.Update(key("esc"))
	require.False(t, m.saveDialogActive)
	require.Equal(t, "", m.saveDialogError)
}

func TestSaveDialog_Enter_EmptyPath(t *testing.T) {
	m := testModel()
	m.saveDialogActive = true
	m.saveStackName = "mystack"
	m.saveFileInput.SetValue("")
	m.Update(key("enter"))
	require.Contains(t, m.saveDialogError, "File path cannot be empty")
	require.True(t, m.saveDialogActive)
}

func TestSaveDialog_EnterWithError_ClearsError(t *testing.T) {
	m := testModel()
	m.saveDialogActive = true
	m.saveDialogError = "previous error"
	m.Update(key("enter"))
	require.Equal(t, "", m.saveDialogError)
	require.True(t, m.saveDialogActive)
}

func TestSaveDialog_F_OpensFileBrowser(t *testing.T) {
	m := testModel()
	m.saveDialogActive = true
	m.saveStackName = "mystack"
	m.saveFileInput.SetValue("mystack.yml")
	cmd := m.Update(key("f"))
	require.False(t, m.saveDialogActive)
	require.True(t, m.fileBrowserActive)
	require.Equal(t, "save", m.fileBrowserContext)
	require.NotNil(t, cmd)
}

func TestStackSavedMsg_ShowsSuccess(t *testing.T) {
	m := testModel()
	m.saveDialogActive = true
	m.Update(stackSavedMsg{Path: "/tmp/mystack.yml"})
	require.False(t, m.saveDialogActive)
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "/tmp/mystack.yml")
}

func TestStackSaveErrorMsg_ReturnsToDialog(t *testing.T) {
	m := testModel()
	m.Update(stackSaveErrorMsg{Err: fmt.Errorf("permission denied")})
	require.True(t, m.saveDialogActive)
	require.Contains(t, m.saveDialogError, "permission denied")
}

func TestConfirmResult_SaveOverwrite_Confirmed(t *testing.T) {
	reconstructed := false
	stackMock := noopStackOps()
	stackMock.reconstructStackComposeFn = func(_ string) (string, error) {
		reconstructed = true
		return "version: '3'", nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	loadStacks(m, fakeStacks("mystack"))
	m.pendingAction = "save-overwrite"
	m.confirmDialog.Visible = true
	m.saveStackName = "mystack"
	m.saveFileInput.SetValue("/tmp/mystack.yml")
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.False(t, m.confirmDialog.Visible)
	require.Equal(t, "", m.pendingAction)
	require.NotNil(t, cmd)
	runCmd(cmd)
	require.True(t, reconstructed)
}

func TestConfirmResult_SaveOverwrite_Cancelled(t *testing.T) {
	m := testModel()
	m.pendingAction = "save-overwrite"
	m.confirmDialog.Visible = true
	m.saveStackName = "mystack"
	m.Update(confirmdialog.ResultMsg{Confirmed: false})
	require.False(t, m.confirmDialog.Visible)
	require.True(t, m.saveDialogActive)
	require.Equal(t, "", m.pendingAction)
}

func TestFileBrowser_Esc_ReturnsSaveDialog(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserContext = "save"
	m.fileBrowserPath = "/tmp"
	m.saveFileInput.SetValue("mystack.yml")
	m.saveStackName = "mystack"
	m.Update(key("esc"))
	require.False(t, m.fileBrowserActive)
	require.True(t, m.saveDialogActive)
	require.Equal(t, "/tmp/mystack.yml", m.saveFileInput.Value())
}

func TestFileBrowser_Esc_ReturnsCreateDialog(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserContext = "create"
	m.Update(key("esc"))
	require.False(t, m.fileBrowserActive)
	require.True(t, m.createDialogActive)
}

func TestSaveStackToFileCmd_Success(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := tmpDir + "/test-stack.yml"

	stackMock := noopStackOps()
	stackMock.reconstructStackComposeFn = func(_ string) (string, error) {
		return "version: '3'\nservices:\n  web:\n    image: nginx\n", nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	cmd := m.saveStackToFileCmd("mystack", filePath)
	msg := runCmd(cmd)
	saved, ok := msg.(stackSavedMsg)
	require.True(t, ok)
	require.Equal(t, filePath, saved.Path)

	// Verify file was written
	data, err := os.ReadFile(filePath)
	require.NoError(t, err)
	require.Contains(t, string(data), "nginx")
}

func TestSaveStackToFileCmd_ReconstructError(t *testing.T) {
	stackMock := noopStackOps()
	stackMock.reconstructStackComposeFn = func(_ string) (string, error) {
		return "", fmt.Errorf("stack not found")
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	cmd := m.saveStackToFileCmd("mystack", "/tmp/test.yml")
	msg := runCmd(cmd)
	errMsg, ok := msg.(stackSaveErrorMsg)
	require.True(t, ok)
	require.Contains(t, errMsg.Err.Error(), "stack not found")
}

func TestSaveStackToFileCmd_WriteError(t *testing.T) {
	stackMock := noopStackOps()
	stackMock.reconstructStackComposeFn = func(_ string) (string, error) {
		return "version: '3'", nil
	}
	m := testModel(func(m *Model) { m.deps.Stacks = stackMock })
	// Use an invalid path to trigger write error
	cmd := m.saveStackToFileCmd("mystack", "/proc/0/nonexistent/test.yml")
	msg := runCmd(cmd)
	errMsg, ok := msg.(stackSaveErrorMsg)
	require.True(t, ok)
	require.Error(t, errMsg.Err)
}

func TestFilesLoadedMsg_Error_SaveContext(t *testing.T) {
	m := testModel()
	m.fileBrowserContext = "save"
	m.Update(filesLoadedMsg{Path: "/nope", Error: fmt.Errorf("no access")})
	require.False(t, m.fileBrowserActive)
	require.True(t, m.saveDialogActive)
	require.Contains(t, m.saveDialogError, "no access")
}

func TestFileBrowser_Enter_SaveHere(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserContext = "save"
	m.fileBrowserPath = "/tmp/mydir"
	m.fileBrowserFiles = []string{"..", saveDirSentinel, "/tmp/mydir/sub/"}
	m.fileBrowserCursor = 1 // [Save here]
	m.saveFileInput.SetValue("mystack.yml")
	m.saveStackName = "mystack"
	m.Update(key("enter"))
	require.False(t, m.fileBrowserActive)
	require.True(t, m.saveDialogActive)
	require.Equal(t, "/tmp/mydir/mystack.yml", m.saveFileInput.Value())
}

func TestFilesLoadedMsg_SaveContext_InjectsSentinel(t *testing.T) {
	m := testModel()
	m.fileBrowserContext = "save"
	m.Update(filesLoadedMsg{Path: "/tmp", Files: []string{"..", "/tmp/sub/", "/tmp/file.yml"}})
	require.True(t, m.fileBrowserActive)
	require.Equal(t, []string{"..", saveDirSentinel, "/tmp/sub/", "/tmp/file.yml"}, m.fileBrowserFiles)
}

func TestFilesLoadedMsg_CreateContext_NoSentinel(t *testing.T) {
	m := testModel()
	m.fileBrowserContext = "create"
	m.Update(filesLoadedMsg{Path: "/tmp", Files: []string{"..", "/tmp/sub/"}})
	require.True(t, m.fileBrowserActive)
	require.Equal(t, []string{"..", "/tmp/sub/"}, m.fileBrowserFiles)
}

func TestFileBrowser_Enter_File_SaveContext(t *testing.T) {
	m := testModel()
	m.fileBrowserActive = true
	m.fileBrowserContext = "save"
	m.fileBrowserFiles = []string{"/tmp/existing-file.yml"}
	m.fileBrowserCursor = 0
	m.saveFileInput.SetValue("mystack.yml")
	m.saveStackName = "mystack"
	m.Update(key("enter"))
	require.False(t, m.fileBrowserActive)
	require.True(t, m.saveDialogActive)
	require.Equal(t, "/tmp/mystack.yml", m.saveFileInput.Value())
}
