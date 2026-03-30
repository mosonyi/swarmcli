package servicesview

import (
	"context"
	"testing"

	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/scaledialog"

	"github.com/stretchr/testify/require"
)

func TestScaleService_Timeout_ReturnsScaleError(t *testing.T) {
	m := testModel(func(m *Model) {
		m.deps.Services = &mockServiceOps{
			scaleServiceFn: func(_ context.Context, _ string, _ uint64) error {
				return context.DeadlineExceeded
			},
			restartServiceFn:    func(_ context.Context, _ string) error { return nil },
			removeServiceFn:     func(_ context.Context, _ string) error { return nil },
			rollbackServiceFn:   func(_ context.Context, _ string) error { return nil },
			loadNodeServicesFn:  func(_ string) []docker.ServiceEntry { return nil },
			loadStackServicesFn: func(_ string) []docker.ServiceEntry { return nil },
		}
	})
	loadServices(m, fakeEntries("web"))
	m.scaleDialog.Visible = true
	cmd := m.Update(scaledialog.ResultMsg{Confirmed: true, Replicas: 3})
	require.NotNil(t, cmd)
	msg := runCmd(cmd)
	_, ok := msg.(ScaleErrorMsg)
	require.True(t, ok, "expected ScaleErrorMsg, got %T", msg)
}

func TestRemoveService_Timeout_ReturnsRemoveError(t *testing.T) {
	m := testModel(func(m *Model) {
		m.deps.Services = &mockServiceOps{
			scaleServiceFn:   func(_ context.Context, _ string, _ uint64) error { return nil },
			restartServiceFn: func(_ context.Context, _ string) error { return nil },
			removeServiceFn: func(_ context.Context, _ string) error {
				return context.DeadlineExceeded
			},
			rollbackServiceFn:   func(_ context.Context, _ string) error { return nil },
			loadNodeServicesFn:  func(_ string) []docker.ServiceEntry { return nil },
			loadStackServicesFn: func(_ string) []docker.ServiceEntry { return nil },
		}
	})
	loadServices(m, fakeEntries("web"))
	m.pendingAction = "remove"
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
	msg := runCmd(cmd)
	_, ok := msg.(RemoveErrorMsg)
	require.True(t, ok, "expected RemoveErrorMsg, got %T", msg)
}

func TestRestartService_Timeout_ReturnsRestartError(t *testing.T) {
	m := testModel(func(m *Model) {
		m.deps.Services = &mockServiceOps{
			scaleServiceFn: func(_ context.Context, _ string, _ uint64) error { return nil },
			restartServiceFn: func(_ context.Context, _ string) error {
				return context.DeadlineExceeded
			},
			removeServiceFn:     func(_ context.Context, _ string) error { return nil },
			rollbackServiceFn:   func(_ context.Context, _ string) error { return nil },
			loadNodeServicesFn:  func(_ string) []docker.ServiceEntry { return nil },
			loadStackServicesFn: func(_ string) []docker.ServiceEntry { return nil },
		}
	})
	loadServices(m, fakeEntries("web"))
	m.pendingAction = "restart"
	m.confirmDialog.Visible = true
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
	msg := runCmd(cmd)
	_, ok := msg.(RestartErrorMsg)
	require.True(t, ok, "expected RestartErrorMsg, got %T", msg)
}
