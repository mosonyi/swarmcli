package configsview

import (
	"context"
	"fmt"
	"testing"

	"swarmcli/docker"
	"swarmcli/views/view"

	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigsCmd_CallsListConfigs(t *testing.T) {
	called := false
	mock := noopConfigOps()
	mock.listConfigsFn = func(_ context.Context) ([]swarm.Config, error) {
		called = true
		return []swarm.Config{
			{ID: "id1", Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "c1"}}},
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.loadConfigsCmd()
	msg := runCmd(cmd)
	require.True(t, called)
	loaded, ok := msg.(configsLoadedMsg)
	require.True(t, ok)
	require.Len(t, loaded, 1)
}

func TestLoadConfigsCmd_Error_ReturnsErrorMsg(t *testing.T) {
	mock := noopConfigOps()
	mock.listConfigsFn = func(_ context.Context) ([]swarm.Config, error) {
		return nil, fmt.Errorf("connection refused")
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.loadConfigsCmd()
	msg := runCmd(cmd)
	_, ok := msg.(errorMsg)
	require.True(t, ok)
}

func TestDeleteConfigCmd_CallsDeleteConfig(t *testing.T) {
	deleted := ""
	mock := noopConfigOps()
	mock.deleteConfigFn = func(_ context.Context, nameOrID string) error {
		deleted = nameOrID
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.deleteConfigCmd("my-config")
	msg := runCmd(cmd)
	del, ok := msg.(configDeletedMsg)
	require.True(t, ok)
	require.Equal(t, "my-config", del.Name)
	require.Equal(t, "my-config", deleted)
}

func TestDeleteConfigCmd_Error_ReturnsErrorMsg(t *testing.T) {
	mock := noopConfigOps()
	mock.deleteConfigFn = func(_ context.Context, _ string) error {
		return fmt.Errorf("not found")
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.deleteConfigCmd("missing")
	msg := runCmd(cmd)
	_, ok := msg.(errorMsg)
	require.True(t, ok)
}

func TestInspectConfigCmd_NavigatesToInspect(t *testing.T) {
	mock := noopConfigOps()
	mock.inspectConfigFn = func(_ context.Context, nameOrID string) (*docker.ConfigWithDecodedData, error) {
		return &docker.ConfigWithDecodedData{
			Config: swarm.Config{
				ID:   "id1",
				Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: nameOrID}},
			},
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.inspectConfigCmd("c1")
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
	payload := nav.Payload.(map[string]any)
	require.Contains(t, payload["title"], "c1")
}

func TestInspectRawConfigCmd_NavigatesToInspect(t *testing.T) {
	mock := noopConfigOps()
	mock.inspectConfigFn = func(_ context.Context, _ string) (*docker.ConfigWithDecodedData, error) {
		return &docker.ConfigWithDecodedData{
			Config: swarm.Config{
				ID:   "id1",
				Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "c1"}},
			},
			Data: []byte("key=value"),
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.inspectRawConfigCmd("c1")
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	payload := nav.Payload.(map[string]any)
	require.Contains(t, payload["title"], "raw")
}

func TestComputeConfigUsedCmd_SetsUsedMap(t *testing.T) {
	mock := noopConfigOps()
	mock.listServicesUsingConfigIDFn = func(_ context.Context, configID string) ([]swarm.Service, error) {
		if configID == "id-used" {
			return []swarm.Service{{ID: "svc1"}}, nil
		}
		return nil, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })

	cfgs := []docker.ConfigWithDecodedData{
		{Config: swarm.Config{ID: "id-used"}},
		{Config: swarm.Config{ID: "id-unused"}},
	}
	cmd := m.computeConfigUsedCmd(cfgs)
	msg := runCmd(cmd)
	usedMap, ok := msg.(usedStatusUpdatedMsg)
	require.True(t, ok)
	require.True(t, usedMap["id-used"])
	require.False(t, usedMap["id-unused"])
}

func TestGetUsedByStacksCmd_ReturnsUsedByMsg(t *testing.T) {
	mock := noopConfigOps()
	mock.inspectConfigFn = func(_ context.Context, _ string) (*docker.ConfigWithDecodedData, error) {
		return &docker.ConfigWithDecodedData{
			Config: swarm.Config{ID: "cfg-id", Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "c1"}}},
		}, nil
	}
	mock.listServicesUsingConfigNameFn = func(_ context.Context, _ string) ([]swarm.Service, error) {
		return []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "web", Labels: map[string]string{"com.docker.stack.namespace": "mystack"}}}},
		}, nil
	}
	mock.listServicesUsingConfigIDFn = func(_ context.Context, _ string) ([]swarm.Service, error) {
		return nil, nil
	}

	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.getUsedByStacksCmd("c1")
	msg := runCmd(cmd)
	used, ok := msg.(usedByMsg)
	require.True(t, ok)
	require.Equal(t, "c1", used.ConfigName)
	require.Len(t, used.UsedBy, 1)
	require.Equal(t, "mystack", used.UsedBy[0].StackName)
}

func TestCreateConfigFromContentCmd_Success(t *testing.T) {
	created := ""
	mock := noopConfigOps()
	mock.createConfigFn = func(_ context.Context, name string, _ []byte, _ map[string]string) (swarm.Config, error) {
		created = name
		return swarm.Config{ID: "new-id", Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: name}}}, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.createConfigFromContentCmd("myconfig", []byte("data"), nil)
	msg := runCmd(cmd)
	_, ok := msg.(configCreatedMsg)
	require.True(t, ok)
	require.Equal(t, "myconfig", created)
}

func TestCreateConfigFromContentCmd_Error(t *testing.T) {
	mock := noopConfigOps()
	mock.createConfigFn = func(_ context.Context, _ string, _ []byte, _ map[string]string) (swarm.Config, error) {
		return swarm.Config{}, fmt.Errorf("already exists")
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.createConfigFromContentCmd("dup", []byte("data"), nil)
	msg := runCmd(cmd)
	_, ok := msg.(configCreateErrorMsg)
	require.True(t, ok)
}

func TestCheckConfigsCmd_NoChange_ReturnsTick(t *testing.T) {
	mock := noopConfigOps()
	mock.listConfigsFn = func(_ context.Context) ([]swarm.Config, error) {
		return nil, nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	cmd := m.checkConfigsCmd(0)
	msg := runCmd(cmd)
	_, isLoaded := msg.(configsLoadedMsg)
	require.False(t, isLoaded, "should not return configsLoadedMsg when no change")
}

func TestRotateConfigCmd_Success(t *testing.T) {
	rotated := false
	mock := noopConfigOps()
	mock.rotateConfigInServicesFn = func(_ context.Context, _ *swarm.Config, _ swarm.Config) error {
		rotated = true
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Configs = mock })
	oldCfg := &docker.ConfigWithDecodedData{Config: swarm.Config{ID: "old"}}
	newCfg := &docker.ConfigWithDecodedData{Config: swarm.Config{ID: "new", Spec: swarm.ConfigSpec{Annotations: swarm.Annotations{Name: "c1"}}}}
	cmd := m.rotateConfigCmd(oldCfg, newCfg)
	msg := runCmd(cmd)
	_, ok := msg.(configRotatedMsg)
	require.True(t, ok)
	require.True(t, rotated)
}

func TestRotateConfigCmd_NilNewConfig(t *testing.T) {
	m := testModel()
	cmd := m.rotateConfigCmd(nil, nil)
	require.Nil(t, cmd)
}
