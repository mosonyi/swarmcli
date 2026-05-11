// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"context"
	"fmt"
	"testing"
	"time"

	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockNetworkOps struct {
	listNetworksFn             func(ctx context.Context) ([]network.Summary, error)
	inspectNetworkFn           func(ctx context.Context, networkID string) (network.Inspect, error)
	removeNetworkFn            func(ctx context.Context, networkID string) error
	createNetworkFn            func(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error)
	pruneNetworksFn            func(ctx context.Context) (network.PruneReport, error)
	listServicesUsingNetworkFn func(ctx context.Context, networkID, networkName string) ([]string, error)
}

func (m *mockNetworkOps) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	return m.listNetworksFn(ctx)
}
func (m *mockNetworkOps) InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	return m.inspectNetworkFn(ctx, networkID)
}
func (m *mockNetworkOps) RemoveNetwork(ctx context.Context, networkID string) error {
	return m.removeNetworkFn(ctx, networkID)
}
func (m *mockNetworkOps) CreateNetwork(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error) {
	return m.createNetworkFn(ctx, name, opts)
}
func (m *mockNetworkOps) PruneNetworks(ctx context.Context) (network.PruneReport, error) {
	return m.pruneNetworksFn(ctx)
}
func (m *mockNetworkOps) ListServicesUsingNetwork(ctx context.Context, networkID, networkName string) ([]string, error) {
	return m.listServicesUsingNetworkFn(ctx, networkID, networkName)
}

type mockClientOps struct {
	getClientFn   func() (*client.Client, error)
	resetClientFn func()
}

func (m *mockClientOps) GetClient() (*client.Client, error) {
	return m.getClientFn()
}
func (m *mockClientOps) ResetClient() {
	if m.resetClientFn != nil {
		m.resetClientFn()
	}
}

// --- helpers ---

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "shift+tab":
		return tea.KeyMsg{Type: tea.KeyShiftTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "pgup":
		return tea.KeyMsg{Type: tea.KeyPgUp}
	case "pgdown":
		return tea.KeyMsg{Type: tea.KeyPgDown}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case " ":
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	}
	if len(s) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func noopNetworkOps() *mockNetworkOps {
	return &mockNetworkOps{
		listNetworksFn: func(_ context.Context) ([]network.Summary, error) {
			return nil, nil
		},
		inspectNetworkFn: func(_ context.Context, _ string) (network.Inspect, error) {
			return network.Inspect{}, nil
		},
		removeNetworkFn: func(_ context.Context, _ string) error {
			return nil
		},
		createNetworkFn: func(_ context.Context, _ string, _ network.CreateOptions) (string, []string, error) {
			return "", nil, nil
		},
		pruneNetworksFn: func(_ context.Context) (network.PruneReport, error) {
			return network.PruneReport{}, nil
		},
		listServicesUsingNetworkFn: func(_ context.Context, _, _ string) ([]string, error) {
			return nil, nil
		},
	}
}

func noopClientOps() *mockClientOps {
	return &mockClientOps{
		getClientFn: func() (*client.Client, error) {
			return nil, fmt.Errorf("no client in test")
		},
	}
}

func testModel(opts ...func(*Model)) *Model {
	m := New(80, 24)
	m.deps = docker.Deps{Networks: noopNetworkOps(), Client: noopClientOps()}
	for _, o := range opts {
		o(m)
	}
	return m
}

func fakeNetworks(names ...string) []networkItem {
	now := time.Now()
	items := make([]networkItem, len(names))
	for i, name := range names {
		items[i] = networkItem{
			Name:      name,
			ID:        "id-" + name,
			Driver:    "overlay",
			Scope:     "swarm",
			CreatedAt: now,
		}
	}
	return items
}

func loadNetworks(m *Model, items []networkItem) {
	m.Update(NetworksLoadedMsg{Networks: items})
}

// --- Tests ---

func TestNew(t *testing.T) {
	m := New(80, 24)
	require.Equal(t, 80, m.width)
	require.Equal(t, 24, m.height)
	require.Equal(t, stateLoading, m.state)
	require.True(t, m.visible)
	require.Equal(t, SortByName, m.sortField)
	require.True(t, m.sortAscending)
}

func TestName(t *testing.T) {
	m := testModel()
	require.Equal(t, "networks", m.Name())
}

func TestCapturesInput_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.CapturesInput())
}

func TestCapturesInput_ConfirmVisible(t *testing.T) {
	m := testModel()
	m.confirmDialog.Visible = true
	require.True(t, m.CapturesInput())
}

func TestCapturesInput_CreateDialogActive(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	require.True(t, m.CapturesInput())
}

func TestCapturesInput_ErrorDialogActive(t *testing.T) {
	m := testModel()
	m.errorDialogActive = true
	require.True(t, m.CapturesInput())
}

func TestHasActiveFilter_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.HasActiveFilter())
}

func TestIsSearching_Default(t *testing.T) {
	m := testModel()
	require.False(t, m.IsSearching())
}

func TestIsSearching_UsedByView(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	require.True(t, m.IsSearching())
}

func TestIsSearching_CreateDialog(t *testing.T) {
	m := testModel()
	m.createDialogActive = true
	require.True(t, m.IsSearching())
}

func TestHasErrors(t *testing.T) {
	m := testModel()
	require.False(t, m.HasErrors())
}

func TestOnEnter_SetsVisible(t *testing.T) {
	m := testModel()
	m.visible = false
	cmd := m.OnEnter()
	require.True(t, m.visible)
	require.True(t, m.resetCursorOnNextLoad)
	require.NotNil(t, cmd)
}

func TestOnExit_ClearsVisible(t *testing.T) {
	m := testModel()
	m.OnExit()
	require.False(t, m.visible)
}

func TestShortHelpItems_MainView(t *testing.T) {
	m := testModel()
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["c"])
	require.True(t, keys["i"])
	require.True(t, keys["ctrl+d"])
	require.True(t, keys["?"])
}

func TestShortHelpItems_UsedByView(t *testing.T) {
	m := testModel()
	m.usedByViewActive = true
	items := m.ShortHelpItems()
	keys := make(map[string]bool)
	for _, item := range items {
		keys[item.Key] = true
	}
	require.True(t, keys["Enter"])
	require.True(t, keys["Esc"])
}

func TestSetSize(t *testing.T) {
	m := testModel()
	m.SetSize(120, 40)
	require.Equal(t, 120, m.width)
	require.Equal(t, 40, m.height)
	require.Equal(t, 120, m.networksList.Viewport.Width)
	require.Equal(t, 40, m.networksList.Viewport.Height)
}

func TestValidateNetworkName(t *testing.T) {
	require.NoError(t, validateNetworkName("my-network"))
	require.NoError(t, validateNetworkName("net_1.2"))
	require.Error(t, validateNetworkName(""))
	require.Error(t, validateNetworkName("-bad"))
	require.Error(t, validateNetworkName("has space"))
}

func TestValidateSubnet_Valid(t *testing.T) {
	require.NoError(t, validateSubnet("10.0.0.0/24", false))
	require.NoError(t, validateSubnet("fd00::/64", true))
	require.NoError(t, validateSubnet("", false))
}

func TestValidateSubnet_Invalid(t *testing.T) {
	require.Error(t, validateSubnet("not-a-cidr", false))
	require.Error(t, validateSubnet("fd00::/64", false))  // ipv6 in ipv4 slot
	require.Error(t, validateSubnet("10.0.0.0/24", true)) // ipv4 in ipv6 slot
}

func TestValidateGateway_Valid(t *testing.T) {
	require.NoError(t, validateGateway("10.0.0.1", false))
	require.NoError(t, validateGateway("fd00::1", true))
	require.NoError(t, validateGateway("", false))
}

func TestValidateGateway_Invalid(t *testing.T) {
	require.Error(t, validateGateway("not-an-ip", false))
	require.Error(t, validateGateway("fd00::1", false)) // ipv6 in ipv4 slot
	require.Error(t, validateGateway("10.0.0.1", true)) // ipv4 in ipv6 slot
}

func TestTruncateWithEllipsis(t *testing.T) {
	require.Equal(t, "", truncateWithEllipsis("hello", 0))
	require.Equal(t, "…", truncateWithEllipsis("hello", 1))
	require.Equal(t, "he…", truncateWithEllipsis("hello", 3))
	require.Equal(t, "hello", truncateWithEllipsis("hello", 10))
}

func TestGetNetworksHelpContent(t *testing.T) {
	cats := GetNetworksHelpContent()
	require.True(t, len(cats) >= 4)
	require.Equal(t, "General", cats[0].Title)
	require.Equal(t, "Sorting", cats[1].Title)
	require.Equal(t, "Navigation", cats[2].Title)
	require.Equal(t, "Danger Zone", cats[3].Title)
}
