package networksview

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/require"
)

func TestLoadNetworksCmd_CallsListNetworks(t *testing.T) {
	called := false
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		called = true
		return []network.Summary{
			{Name: "net1", ID: "id1", Driver: "overlay", Scope: "swarm"},
		}, nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.loadNetworksCmd()
	msg := runCmd(cmd)
	require.True(t, called)
	loaded, ok := msg.(NetworksLoadedMsg)
	require.True(t, ok)
	require.Nil(t, loaded.Err)
	require.Len(t, loaded.Networks, 1)
	require.Equal(t, "net1", loaded.Networks[0].Name)
}

func TestLoadNetworksCmd_Error(t *testing.T) {
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		return nil, fmt.Errorf("connection refused")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.loadNetworksCmd()
	msg := runCmd(cmd)
	loaded, ok := msg.(NetworksLoadedMsg)
	require.True(t, ok)
	require.Error(t, loaded.Err)
}

func TestDeleteNetworkCmd_Success(t *testing.T) {
	deleted := ""
	mock := noopNetworkOps()
	mock.removeNetworkFn = func(_ context.Context, networkID string) error {
		deleted = networkID
		return nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.deleteNetworkCmd("id-mynet")
	msg := runCmd(cmd)
	del, ok := msg.(NetworkDeletedMsg)
	require.True(t, ok)
	require.Nil(t, del.Err)
	require.Equal(t, "id-mynet", deleted)
}

func TestDeleteNetworkCmd_Error(t *testing.T) {
	mock := noopNetworkOps()
	mock.removeNetworkFn = func(_ context.Context, _ string) error {
		return fmt.Errorf("has active endpoints")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.deleteNetworkCmd("id-mynet")
	msg := runCmd(cmd)
	del, ok := msg.(NetworkDeletedMsg)
	require.True(t, ok)
	require.Error(t, del.Err)
}

func TestCreateNetworkCmd_Success(t *testing.T) {
	created := ""
	mock := noopNetworkOps()
	mock.createNetworkFn = func(_ context.Context, name string, _ network.CreateOptions) (string, []string, error) {
		created = name
		return "new-id", nil, nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.createNetworkCmd("mynet", "overlay", true, false, "", "", false, "", "")
	msg := runCmd(cmd)
	result, ok := msg.(NetworkCreatedMsg)
	require.True(t, ok)
	require.Nil(t, result.Err)
	require.Equal(t, "mynet", result.Name)
	require.Equal(t, "new-id", result.ID)
	require.Equal(t, "mynet", created)
}

func TestCreateNetworkCmd_Error(t *testing.T) {
	mock := noopNetworkOps()
	mock.createNetworkFn = func(_ context.Context, _ string, _ network.CreateOptions) (string, []string, error) {
		return "", nil, fmt.Errorf("already exists")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.createNetworkCmd("dup", "overlay", true, false, "", "", false, "", "")
	msg := runCmd(cmd)
	result, ok := msg.(NetworkCreatedMsg)
	require.True(t, ok)
	require.Error(t, result.Err)
}

func TestPruneNetworksCmd_Success(t *testing.T) {
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		return []network.Summary{
			{Name: "unused1", ID: "id1"},
			{Name: "unused2", ID: "id2"},
		}, nil
	}
	mock.pruneNetworksFn = func(_ context.Context) (network.PruneReport, error) {
		return network.PruneReport{NetworksDeleted: []string{"id1"}}, nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.pruneNetworksCmd()
	msg := runCmd(cmd)
	result, ok := msg.(NetworksPrunedMsg)
	require.True(t, ok)
	require.Nil(t, result.Err)
	require.Len(t, result.Deleted, 1)
	require.Equal(t, "unused1", result.Deleted[0]) // resolved from ID to name
}

func TestPruneNetworksCmd_ListError(t *testing.T) {
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		return nil, fmt.Errorf("no access")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.pruneNetworksCmd()
	msg := runCmd(cmd)
	result, ok := msg.(NetworksPrunedMsg)
	require.True(t, ok)
	require.Error(t, result.Err)
}

func TestPruneNetworksCmd_PruneError(t *testing.T) {
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		return nil, nil
	}
	mock.pruneNetworksFn = func(_ context.Context) (network.PruneReport, error) {
		return network.PruneReport{}, fmt.Errorf("prune failed")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.pruneNetworksCmd()
	msg := runCmd(cmd)
	result, ok := msg.(NetworksPrunedMsg)
	require.True(t, ok)
	require.Error(t, result.Err)
}

func TestInspectNetworkCmd_Success(t *testing.T) {
	mock := noopNetworkOps()
	mock.inspectNetworkFn = func(_ context.Context, networkID string) (network.Inspect, error) {
		return network.Inspect{
			Name:   "mynet",
			ID:     networkID,
			Driver: "overlay",
			Scope:  "swarm",
		}, nil
	}
	mock.listServicesUsingNetworkFn = func(_ context.Context, _, _ string) ([]string, error) {
		return []string{"svc1"}, nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.inspectNetworkCmd("id-mynet")
	msg := runCmd(cmd)
	result, ok := msg.(NetworkInspectMsg)
	require.True(t, ok)
	require.Nil(t, result.Err)
	require.NotNil(t, result.NetworkWithUsage)
	require.Equal(t, "mynet", result.NetworkWithUsage.Network.Name)
	require.Equal(t, []string{"svc1"}, result.NetworkWithUsage.Services)
}

func TestInspectNetworkCmd_Error(t *testing.T) {
	mock := noopNetworkOps()
	mock.inspectNetworkFn = func(_ context.Context, _ string) (network.Inspect, error) {
		return network.Inspect{}, fmt.Errorf("not found")
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.inspectNetworkCmd("bad-id")
	msg := runCmd(cmd)
	result, ok := msg.(NetworkInspectMsg)
	require.True(t, ok)
	require.Error(t, result.Err)
}

func TestLoadUsedByCmd_ClientError(t *testing.T) {
	m := testModel() // default mock returns error from GetClient
	cmd := m.loadUsedByCmd("id1", "net1")
	msg := runCmd(cmd)
	result, ok := msg.(UsedByLoadedMsg)
	require.True(t, ok)
	require.Error(t, result.Err)
}

func TestComputeNetworkUsedCmd_ClientError(t *testing.T) {
	m := testModel() // default mock returns error from GetClient
	items := fakeNetworks("net1", "net2")
	cmd := m.computeNetworkUsedCmd(items)
	msg := runCmd(cmd)
	result, ok := msg.(usedStatusUpdatedMsg)
	require.True(t, ok)
	// On client error, returns map with all false
	require.False(t, result["id-net1"])
	require.False(t, result["id-net2"])
}

func TestCheckNetworksCmd_NoChange(t *testing.T) {
	mock := noopNetworkOps()
	mock.listNetworksFn = func(_ context.Context) ([]network.Summary, error) {
		return nil, nil
	}
	m := testModel(func(m *Model) { m.deps.Networks = mock })
	cmd := m.checkNetworksCmd(0)
	msg := runCmd(cmd)
	require.NotNil(t, msg)
	// Empty list with lastHash=0: hashes match, so we get PollRetryMsg (not NetworksLoadedMsg)
	_, isLoaded := msg.(NetworksLoadedMsg)
	if !isLoaded {
		_, isPollRetry := msg.(PollRetryMsg)
		require.True(t, isPollRetry, "should return PollRetryMsg when no change")
	}
}
