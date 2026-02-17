package systeminfoview

import (
	"testing"

	"swarmcli/docker"

	"github.com/stretchr/testify/require"
)

// --- mock ---

type mockClusterInfoOps struct {
	getCurrentContextFn   func() (string, error)
	getContainerCountFn   func() (int, error)
	getServiceCountFn     func() (int, error)
	getSwarmCPUCapacityFn func() (float64, error)
	getSwarmMemCapacityFn func() (int64, error)
	getSwarmCPUUsageFn    func() (string, error)
	getSwarmMemUsageFn    func() (string, error)
	getDockerVersionFn    func() (string, error)
}

func (m *mockClusterInfoOps) GetCurrentContext() (string, error) {
	return m.getCurrentContextFn()
}
func (m *mockClusterInfoOps) GetContainerCount() (int, error) {
	return m.getContainerCountFn()
}
func (m *mockClusterInfoOps) GetServiceCount() (int, error) {
	return m.getServiceCountFn()
}
func (m *mockClusterInfoOps) GetSwarmCPUCapacity() (float64, error) {
	return m.getSwarmCPUCapacityFn()
}
func (m *mockClusterInfoOps) GetSwarmMemCapacity() (int64, error) {
	return m.getSwarmMemCapacityFn()
}
func (m *mockClusterInfoOps) GetSwarmCPUUsage() (string, error) {
	return m.getSwarmCPUUsageFn()
}
func (m *mockClusterInfoOps) GetSwarmMemUsage() (string, error) {
	return m.getSwarmMemUsageFn()
}
func (m *mockClusterInfoOps) GetDockerVersion() (string, error) {
	return m.getDockerVersionFn()
}

var _ docker.ClusterInfoOps = (*mockClusterInfoOps)(nil)

func noopClusterInfoOps() *mockClusterInfoOps {
	return &mockClusterInfoOps{
		getCurrentContextFn:   func() (string, error) { return "default", nil },
		getContainerCountFn:   func() (int, error) { return 5, nil },
		getServiceCountFn:     func() (int, error) { return 3, nil },
		getSwarmCPUCapacityFn: func() (float64, error) { return 4.0, nil },
		getSwarmMemCapacityFn: func() (int64, error) { return 8 * 1024 * 1024 * 1024, nil },
		getSwarmCPUUsageFn:    func() (string, error) { return "12.5%", nil },
		getSwarmMemUsageFn:    func() (string, error) { return "45.3%", nil },
		getDockerVersionFn:    func() (string, error) { return "27.0.0", nil },
	}
}

func testDeps() docker.Deps {
	return docker.Deps{ClusterInfo: noopClusterInfoOps()}
}

// --- Tests ---

func TestNew(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	require.NotNil(t, m)
	require.Equal(t, "1.0.0", m.version)
	require.Equal(t, "default", m.context)
	require.True(t, m.firstLoad)
}

func TestLoadStatus(t *testing.T) {
	ops := noopClusterInfoOps()
	ops.getCurrentContextFn = func() (string, error) { return "test-ctx", nil }
	ops.getContainerCountFn = func() (int, error) { return 10, nil }
	ops.getServiceCountFn = func() (int, error) { return 5, nil }
	m := New(docker.Deps{ClusterInfo: ops}, "1.0.0")
	cmd := m.LoadStatus()
	msg := cmd()
	statusMsg, ok := msg.(Msg)
	require.True(t, ok)
	require.Equal(t, "test-ctx", statusMsg.context)
	require.Equal(t, 10, statusMsg.containers)
	require.Equal(t, 5, statusMsg.services)
}

func TestLoadSlowStatus(t *testing.T) {
	ops := noopClusterInfoOps()
	ops.getSwarmCPUUsageFn = func() (string, error) { return "25.0%", nil }
	ops.getSwarmMemUsageFn = func() (string, error) { return "50.0%", nil }
	m := New(docker.Deps{ClusterInfo: ops}, "1.0.0")
	cmd := m.LoadSlowStatus()
	msg := cmd()
	slow, ok := msg.(SlowStatusMsg)
	require.True(t, ok)
	require.Equal(t, "25.0%", slow.cpu)
	require.Equal(t, "50.0%", slow.mem)
}

func TestUpdate_Msg(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	cmd := m.Update(Msg{
		context:    "ctx",
		cpu:        "10%",
		mem:        "20%",
		containers: 3,
		services:   2,
	})
	require.Equal(t, "ctx", m.context)
	require.Equal(t, 3, m.containerCount)
	require.Equal(t, 2, m.serviceCount)
	require.NotNil(t, cmd) // triggers LoadSlowStatus
}

func TestUpdate_SlowStatusMsg(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	cmd := m.Update(SlowStatusMsg{cpu: "15.0%", mem: "30.0%"})
	require.Equal(t, "15.0%", m.cpuUsage)
	require.Equal(t, "30.0%", m.memUsage)
	require.False(t, m.loadingCPU)
	require.False(t, m.loadingMem)
	require.False(t, m.firstLoad)
	require.NotNil(t, cmd) // tick cmd
}

func TestUpdate_TickMsg(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	cmd := m.Update(TickMsg{})
	require.NotNil(t, cmd) // LoadSlowStatus
}

func TestUpdate_SpinnerTickMsg(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	m.loadingCPU = true
	oldSpinner := m.spinner
	cmd := m.Update(SpinnerTickMsg{})
	require.Equal(t, oldSpinner+1, m.spinner)
	require.NotNil(t, cmd) // next spinner tick
}

func TestContent_Format(t *testing.T) {
	result := content("default", "1.0.0", "10%", "20%", 5, 3)
	require.Contains(t, result, "default")
	require.Contains(t, result, "1.0.0")
	require.Contains(t, result, "10%")
	require.Contains(t, result, "20%")
	require.Contains(t, result, "5")
	require.Contains(t, result, "3")
}

func TestUpdateCPUMem_TrendArrows(t *testing.T) {
	m := New(testDeps(), "1.0.0")
	// First update sets baseline
	m.Update(SlowStatusMsg{cpu: "10.0%", mem: "20.0%"})
	require.Equal(t, float64(10), m.prevCPU)
	require.Equal(t, float64(20), m.prevMem)

	// Second update should detect increase
	m.Update(SlowStatusMsg{cpu: "15.0%", mem: "25.0%"})
	require.Contains(t, m.cpuUsage, "15.0%")
	require.Contains(t, m.memUsage, "25.0%")
	require.Equal(t, "up", m.prevCPUTrend)
	require.Equal(t, "up", m.prevMemTrend)
}
