package logsview

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"swarmcli/docker"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// StartStreamingCmd returns a tea.Cmd that starts streaming logs for the given service
// using the provided docker client. It reads the last `tail` lines and then follows.
// - cli: Docker client
// - service: your ServiceEntry (we use ServiceID)
// - tail: number of lines to request as initial history (0 means all)
// - MaxLines: the maximum number of lines to keep in memory (circular buffer behavior)
func StartStreamingCmd(ctx context.Context, service docker.ServiceEntry, tail int, maxLines int) tea.Cmd {
	cli, _ := docker.GetClient()

	return func() tea.Msg {
		lines := make(chan string, 512)
		errs := make(chan error, 1)

		go func() {
			defer close(lines)
			defer close(errs)

			// prepare the logs options using container.LogsOptions (ServiceLogs expects this)
			opts := container.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
				Timestamps: false,
				Details:    true, // Include task and node information in log prefix
			}
			if tail > 0 {
				opts.Tail = fmt.Sprintf("%d", tail)
			} else {
				opts.Tail = "all"
			}
			l().Debugf("[logsview] requesting service logs with Tail=%s", opts.Tail)

			// call ServiceLogs (streams a multiplexed stream)
			reader, err := cli.ServiceLogs(ctx, service.ServiceID, opts)
			if err != nil {
				l().With("service", service.ServiceID).Errorf("ServiceLogs error: %v", err)
				errs <- err
				return
			}
			defer func() { _ = reader.Close() }()

			// demultiplex with stdcopy into pipes
			stdoutR, stdoutW := io.Pipe()
			stderrR, stderrW := io.Pipe()

			var scErr error
			var scWG sync.WaitGroup
			scWG.Add(1)
			go func() {
				defer scWG.Done()
				_, scErr = stdcopy.StdCopy(stdoutW, stderrW, reader)
				_ = stdoutW.Close()
				_ = stderrW.Close()
			}()

			// start scanners that push complete lines into the lines channel
			var wg sync.WaitGroup
			pushScanner := func(r io.Reader) {
				defer wg.Done()
				sc := bufio.NewScanner(r)
				for sc.Scan() {
					line := sc.Text()
					// Format the log line with node information
					formattedLine, nodeName := formatLogLineWithNode(service.ServiceName, line)
					// Store both the formatted line and node name (separated by a special marker)
					// Format: "NODENAME\x00formatted_line" where \x00 is a null byte separator
					lines <- nodeName + "\x00" + formattedLine
				}
			}

			wg.Add(2)
			go pushScanner(stdoutR)
			go pushScanner(stderrR)

			// wait for scanners + stdcopy to finish
			wg.Wait()
			scWG.Wait()

			if scErr != nil {
				level := l().With("service", service.ServiceID)
				if errors.Is(scErr, context.Canceled) {
					level.Debug("log stream closed normally (context canceled)")
				} else {
					level.Warnf("stdcopy finished with error: %v", scErr)
				}
				return
			}
		}()

		// return InitStreamMsg carrying the channels AND the requested MaxLines
		return InitStreamMsg{
			Lines:    lines,
			Errs:     errs,
			MaxLines: maxLines,
		}
	}
}

// StopStreamingCmd returns a cmd that cancels the streaming context (if set on model).
// Use this to stop the docker log stream (kills follow).
func (m *Model) StopStreamingCmd() tea.Cmd {
	return func() tea.Msg {
		m.streamMu.Lock()
		defer m.streamMu.Unlock()
		if m.StreamCancel != nil {
			l().Debugf("[logsview] stop streaming requested")
			m.StreamCancel()
			m.StreamCancel = nil
			m.streamActive = false
		}
		return nil
	}
}

// formatLogLineWithNode parses the Docker log details and formats the line with node information
// Input format: "com.docker.swarm.node.id=xxx,com.docker.swarm.task.id=yyy actual log message"
// Output format: formatted line and node name for filtering
// Returns: ("service_name.task_id@node_name | actual log message", "node_name")
func formatLogLineWithNode(serviceName string, line string) (string, string) {
	// Check if line has Docker details prefix
	if !strings.Contains(line, "com.docker.swarm.") {
		return line, ""
	}

	// Split on first space to separate details from message
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return line, ""
	}

	details := parts[0]
	message := parts[1]

	// Extract node ID and task ID from details
	var nodeID, taskID string

	// Parse key=value pairs
	pairs := strings.Split(details, ",")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			switch kv[0] {
			case "com.docker.swarm.node.id":
				nodeID = kv[1]
			case "com.docker.swarm.task.id":
				taskID = kv[1]
			}
		}
	}

	// Get node hostname from node ID
	nodeName := getNodeHostname(nodeID)
	if nodeName == "" {
		nodeName = nodeID[:12] // fallback to short ID
	}

	// Format task ID (show first 12 chars)
	taskIDShort := taskID
	if len(taskID) > 12 {
		taskIDShort = taskID[:12]
	}

	// Build the formatted prefix with blue color (117 is the light blue we use elsewhere)
	// ANSI escape: \033[38;5;117m for foreground color 117, \033[0m to reset
	prefix := fmt.Sprintf("\033[38;5;117m%s.%s@%s\033[0m", serviceName, taskIDShort, nodeName)

	return fmt.Sprintf("%s | %s", prefix, message), nodeName
}

// getNodeHostname retrieves the hostname for a node ID from the snapshot
func getNodeHostname(nodeID string) string {
	snap := docker.GetSnapshot()
	if snap == nil {
		return ""
	}

	for _, node := range snap.Nodes {
		if node.ID == nodeID {
			if node.Description.Hostname != "" {
				return node.Description.Hostname
			}
		}
	}
	return ""
}
