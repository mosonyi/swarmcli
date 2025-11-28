package systeminfoview

import (
	"time"

	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"swarmcli/docker"
)

type Model struct {
	// We don't need a viewport here, as we will use a fixed size for the content.
	content string

	version string

	context        string
	cpuUsage       string
	memUsage       string
	containerCount int
	serviceCount   int

	// For tracking trends
	prevCPU      float64
	prevMem      float64
	lastUpdate   time.Time
	updateInterval time.Duration
	
	// Loading state
	loadingCPU bool
	loadingMem bool
	spinner    int
	firstLoad  bool
	
	// Trend arrow state
	prevCPUTrend string // "up", "down", or ""
	prevMemTrend string
	cpuBlinkCount int
	memBlinkCount int
}

// Create a new instance
func New(version string) *Model {
	// Get initial context synchronously to display immediately
	context, _ := docker.GetCurrentContext()
	return &Model{
		content:        content(context, version, "", "", 0, 0),
		version:        version,
		context:        context,
		updateInterval: 8 * time.Second,
		lastUpdate:     time.Now(),
		loadingCPU:     true,
		loadingMem:     true,
		firstLoad:      true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), m.spinnerTickCmd())
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func LoadStatus() tea.Cmd {
	return func() tea.Msg {
		// Get fast values immediately
		context, _ := docker.GetCurrentContext()
		containers, _ := docker.GetContainerCount()
		services, _ := docker.GetServiceCount()
		
		// Return immediately with fast values, spinner marker for CPU/MEM
		// Using first frame of spinner charset 14 as marker
		spinnerMarker := spinner.CharSets[14][0]
		return Msg{
			context:    context,
			cpu:        spinnerMarker,
			mem:        spinnerMarker,
			containers: containers,
			services:   services,
		}
	}
}

func LoadSlowStatus() tea.Cmd {
	return func() tea.Msg {
		// Get CPU/MEM - these are slow
		cpu, err := docker.GetSwarmCPUUsage()
		if err != nil {
			cpu = "N/A"
		}
		if cpu == "" {
			cpu = "0.0%"
		}
		
		mem, err := docker.GetSwarmMemUsage()
		if err != nil {
			mem = "N/A"
		}
		if mem == "" {
			mem = "0.0%"
		}
		
		// Return only CPU/MEM update
		return SlowStatusMsg{
			cpu: cpu,
			mem: mem,
		}
	}
}
