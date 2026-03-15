// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package systeminfoview

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"swarmcli/docker"

	"github.com/briandowns/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/mod/semver"
)

type Model struct {
	deps docker.Deps

	// We don't need a viewport here, as we will use a fixed size for the content.
	content string

	version         string
	edition         string
	latest          string
	versionCheckURL string

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

const (
	defaultEdition         = "ce"
	defaultVersionCheckURL = "https://swarmcli.io/api/v1/version"
	versionCheckDisableEnv = "SWARMCLI_DISABLE_VERSION_CHECK"
	versionCheckTimeout    = 4 * time.Second
)

type versionCheckRequest struct {
	Version string `json:"version"`
	Edition string `json:"edition"`
}

type versionCheckResponse struct {
	LatestVersion string `json:"latestVersion"`
}

// Create a new instance
func New(deps docker.Deps, version, edition string) *Model {
	// Get initial context synchronously to display immediately
	context, _ := deps.ClusterInfo.GetCurrentContext()
	normalizedEdition := normalizeEdition(edition)

	return &Model{
		deps:            deps,
		content:         content(context, version, "", "", 0, 0),
		version:         version,
		edition:         normalizedEdition,
		versionCheckURL: defaultVersionCheckURL,
		context:         context,
		updateInterval:  8 * time.Second,
		lastUpdate:      time.Now(),
		loadingCPU:      true,
		loadingMem:      true,
		firstLoad:       true,
	}
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tickCmd(), m.spinnerTickCmd()}
	if cmd := m.CheckLatestVersion(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m *Model) CheckLatestVersion() tea.Cmd {
	currentVersion := strings.TrimSpace(m.version)
	currentEdition := normalizeEdition(m.edition)
	if versionCheckDisabled() {
		l().Infow("startup version check disabled", "env", versionCheckDisableEnv)
		return nil
	}

	return func() tea.Msg {
		latestVersion, err := fetchLatestVersion(m.versionCheckURL, currentVersion, currentEdition)
		if err != nil {
			l().Infow("startup version check failed", "version", currentVersion, "edition", currentEdition, "error", err)
			return NoVersionUpdateMsg{}
		}

		if !shouldShowLatestVersion(currentVersion, latestVersion) {
			return NoVersionUpdateMsg{}
		}

		return LatestVersionMsg{
			latestVersion: latestVersion,
		}
	}
}

func normalizeEdition(edition string) string {
	normalizedEdition := strings.TrimSpace(strings.ToLower(edition))
	if normalizedEdition == "" {
		return defaultEdition
	}

	return normalizedEdition
}

func versionCheckDisabled() bool {
	disabled, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(versionCheckDisableEnv)))
	return err == nil && disabled
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
	currentOK := isSemver(currentVersion)
	latestOK := isSemver(latestVersion)
	if !currentOK && latestOK {
		return true
	}

	return false
}

func fetchLatestVersion(checkURL, currentVersion, edition string) (string, error) {
	edition = normalizeEdition(edition)
	checkURL = strings.TrimSpace(checkURL)
	if checkURL == "" {
		return "", fmt.Errorf("version API URL is empty")
	}

	payload := versionCheckRequest{
		Version: currentVersion,
		Edition: edition,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal version payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, checkURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build version request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: versionCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send version request: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			l().Debugw("version response body close failed", "error", closeErr)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("version API status: %s", resp.Status)
	}

	var decoded versionCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("decode version response: %w", err)
	}

	return strings.TrimSpace(decoded.LatestVersion), nil
}

func isNewerVersion(currentVersion, latestVersion string) bool {
	cmp, ok := compareVersionStrings(latestVersion, currentVersion)
	return ok && cmp > 0
}

func compareVersionStrings(a, b string) (int, bool) {
	normalizedA, ok := normalizeSemver(a)
	if !ok {
		return 0, false
	}
	normalizedB, ok := normalizeSemver(b)
	if !ok {
		return 0, false
	}

	return semver.Compare(normalizedA, normalizedB), true
}

func isSemver(version string) bool {
	_, ok := normalizeSemver(version)
	return ok
}

func normalizeSemver(version string) (string, bool) {
	normalized := strings.TrimSpace(version)
	if normalized == "" {
		return "", false
	}

	if strings.HasPrefix(normalized, "V") {
		normalized = "v" + normalized[1:]
	} else if !strings.HasPrefix(normalized, "v") {
		normalized = "v" + normalized
	}

	if !semver.IsValid(normalized) {
		return "", false
	}

	return normalized, true
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
