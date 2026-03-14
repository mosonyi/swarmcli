// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"swarmcli/docker"

	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	deps docker.Deps

	// We don't need a viewport here, as we will use a fixed size for the content.
	content string

	version string
	edition string
	latest  string
	message string

	context        string
	cpuUsage       string
	memUsage       string
	cpuCapacity    string // Total CPU cores
	memCapacity    string // Total memory
	containerCount int
	serviceCount   int

	// For tracking trends
	prevCPU        float64
	prevMem        float64
	lastUpdate     time.Time
	updateInterval time.Duration

	// Loading state
	loadingCPU bool
	loadingMem bool
	spinner    int
	firstLoad  bool

	// Trend arrow state
	prevCPUTrend  string // "up", "down", or ""
	prevMemTrend  string
	cpuBlinkCount int
	memBlinkCount int
}

var versionCheckURL = "https://swarmcli.io/api/v1/version"

const (
	defaultEdition      = "ce"
	versionCheckTimeout = 4 * time.Second
)

type versionCheckRequest struct {
	Version string `json:"version"`
	Edition string `json:"edition"`
}

type versionCheckResponse struct {
	LatestVersion string `json:"latestVersion"`
	Message       string `json:"message"`
}

// Create a new instance
func New(deps docker.Deps, version, edition string) *Model {
	// Get initial context synchronously to display immediately
	context, _ := deps.ClusterInfo.GetCurrentContext()
	normalizedEdition := strings.TrimSpace(edition)
	if normalizedEdition == "" {
		normalizedEdition = defaultEdition
	}

	return &Model{
		deps:           deps,
		content:        content(context, version, "", "", 0, 0),
		version:        version,
		edition:        normalizedEdition,
		context:        context,
		updateInterval: 8 * time.Second,
		lastUpdate:     time.Now(),
		loadingCPU:     true,
		loadingMem:     true,
		firstLoad:      true,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.tickCmd(), m.spinnerTickCmd(), m.CheckLatestVersion())
}

func (m *Model) CheckLatestVersion() tea.Cmd {
	currentVersion := strings.TrimSpace(m.version)
	currentEdition := strings.TrimSpace(m.edition)
	if currentEdition == "" {
		currentEdition = defaultEdition
	}

	return func() tea.Msg {
		latestVersion, message, err := fetchLatestVersion(currentVersion, currentEdition)
		if err != nil {
			l().Infow("startup version check failed", "version", currentVersion, "edition", currentEdition, "error", err)
			return nil
		}

		if !shouldShowLatestVersion(currentVersion, latestVersion) {
			return nil
		}

		return LatestVersionMsg{
			latestVersion: latestVersion,
			message:       message,
		}
	}
}

func shouldShowLatestVersion(currentVersion, latestVersion string) bool {
	latestVersion = strings.TrimSpace(latestVersion)
	if latestVersion == "" {
		return false
	}

	if isNewerVersion(currentVersion, latestVersion) {
		return true
	}

	// For dev/non-semver builds, still surface latest stable release.
	_, currentOK := parseVersion(currentVersion)
	_, latestOK := parseVersion(latestVersion)
	if !currentOK && latestOK {
		return true
	}

	return false
}

func fetchLatestVersion(currentVersion, edition string) (string, string, error) {
	edition = strings.TrimSpace(edition)
	if edition == "" {
		edition = defaultEdition
	}

	payload := versionCheckRequest{
		Version: currentVersion,
		Edition: edition,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("marshal version payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, versionCheckURL, bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("build version request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: versionCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("send version request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			l().Debugw("version response body close failed", "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("version API status: %s", resp.Status)
	}

	var decoded versionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", "", fmt.Errorf("decode version response: %w", err)
	}

	return strings.TrimSpace(decoded.LatestVersion), strings.TrimSpace(decoded.Message), nil
}

func isNewerVersion(currentVersion, latestVersion string) bool {
	cmp, ok := compareVersionStrings(latestVersion, currentVersion)
	return ok && cmp > 0
}

func compareVersionStrings(a, b string) (int, bool) {
	aParts, ok := parseVersion(a)
	if !ok {
		return 0, false
	}
	bParts, ok := parseVersion(b)
	if !ok {
		return 0, false
	}

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var av, bv int
		if i < len(aParts) {
			av = aParts[i]
		}
		if i < len(bParts) {
			bv = bParts[i]
		}

		if av > bv {
			return 1, true
		}
		if av < bv {
			return -1, true
		}
	}

	return 0, true
}

func parseVersion(version string) ([]int, bool) {
	normalized := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(version), "v"))
	if normalized == "" {
		return nil, false
	}

	if idx := strings.IndexByte(normalized, '-'); idx >= 0 {
		normalized = normalized[:idx]
	}

	parts := strings.Split(normalized, ".")
	if len(parts) == 0 {
		return nil, false
	}

	parsed := make([]int, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			return nil, false
		}

		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		parsed = append(parsed, n)
	}

	return parsed, true
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

func (m *Model) LoadStatus() tea.Cmd {
	clusterInfo := m.deps.ClusterInfo
	return func() tea.Msg {
		// Get fast values immediately
		context, _ := clusterInfo.GetCurrentContext()
		containers, _ := clusterInfo.GetContainerCount()
		services, _ := clusterInfo.GetServiceCount()

		// Get capacity (fast) - show immediately
		cpuCapacity, _ := clusterInfo.GetSwarmCPUCapacity()
		memCapacity, _ := clusterInfo.GetSwarmMemCapacity()

		cpuCapStr := ""
		if cpuCapacity > 0 {
			cpuCapStr = fmt.Sprintf("%.0f cores", cpuCapacity)
		} else {
			cpuCapStr = "-- cores"
		}

		memCapStr := ""
		if memCapacity > 0 {
			memCapStr = fmt.Sprintf("%.0f GB", float64(memCapacity)/1024/1024/1024)
		} else {
			memCapStr = "--- GB"
		}

		// Return immediately with fast values, spinner marker for CPU/MEM usage
		// Using first frame of spinner charset 14 as marker
		spinnerMarker := spinner.CharSets[14][0]
		return Msg{
			context:     context,
			cpu:         spinnerMarker,
			mem:         spinnerMarker,
			cpuCapacity: cpuCapStr,
			memCapacity: memCapStr,
			containers:  containers,
			services:    services,
		}
	}
}

func (m *Model) LoadSlowStatus() tea.Cmd {
	clusterInfo := m.deps.ClusterInfo
	return func() tea.Msg {
		l().Info("LoadSlowStatus: Starting background stats collection")

		// Get CPU/MEM - these are slow
		cpu, err := clusterInfo.GetSwarmCPUUsage()
		if err != nil {
			l().Error("LoadSlowStatus: GetSwarmCPUUsage failed: %v", err)
			cpu = "N/A"
		}
		if cpu == "" {
			cpu = "0.0%"
		}
		l().Info("LoadSlowStatus: CPU usage collected: %s", cpu)

		mem, err := clusterInfo.GetSwarmMemUsage()
		if err != nil {
			l().Error("LoadSlowStatus: GetSwarmMemUsage failed: %v", err)
			mem = "N/A"
		}
		if mem == "" {
			mem = "0.0%"
		}
		l().Info("LoadSlowStatus: Memory usage collected: %s", mem)

		// Return only CPU/MEM update
		return SlowStatusMsg{
			cpu: cpu,
			mem: mem,
		}
	}
}
