// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"io"
	"os/exec"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/pkg/stdcopy"
)

// startStreamingCmd returns a tea.Cmd that starts streaming logs for the given service
// using the provided docker client. It reads the last `tail` lines and then follows.
// - cli: Docker client
// - service: your ServiceEntry (we use ServiceID)
// - tail: number of lines to request as initial history (0 means all)
// - MaxLines: the maximum number of lines to keep in memory (circular buffer behavior)
func (m *Model) startStreamingCmd(ctx context.Context, service docker.ServiceEntry, tail int, maxLines int) tea.Cmd {
	clientOps := m.deps.Client
	cli, err := clientOps.GetClient()
	if err != nil {
		return func() tea.Msg {
			return StreamErrMsg{Err: fmt.Errorf("failed to get docker client: %w", err)}
		}
	}

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
				// NOTE: This matches a Docker daemon error string. Update if Docker changes the message.
				if strings.Contains(strings.ToLower(err.Error()), "tty service logs only supported with --raw") {
					l().With("service", service.ServiceID).Warn("ServiceLogs requires --raw for tty service; falling back to docker CLI")
					if cliErr := m.streamServiceLogsRawCLI(ctx, service, opts.Tail, lines); cliErr != nil {
						errs <- cliErr
					}
					return
				}
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
					formattedLine, nodeName, taskID := m.formatLogLineWithNode(service.ServiceName, line)
					// Store the formatted line, node name and task ID (separated by a special marker)
					// Format: "NODENAME\x00TASKID\x00formatted_line" where \x00 is a null byte separator
					select {
					case lines <- nodeName + "\x00" + taskID + "\x00" + formattedLine:
					case <-ctx.Done():
						return
					}
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
				} else if shouldFallbackToRawFromStdCopy(scErr) {
					level.Warnf("stdcopy failed (%v); falling back to docker CLI raw logs", scErr)
					if cliErr := m.streamServiceLogsRawCLI(ctx, service, opts.Tail, lines); cliErr != nil {
						errs <- cliErr
					}
				} else {
					level.Warnf("stdcopy finished with error: %v", scErr)
					errs <- scErr
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

func (m *Model) streamServiceLogsRawCLI(ctx context.Context, service docker.ServiceEntry, tail string, lines chan<- string) error {
	ctxName, err := docker.GetContextFromEnv()
	if err != nil {
		return fmt.Errorf("failed to determine docker context: %w", err)
	}

	args := docker.BuildRawLogArgs(ctxName, service.ServiceID, "--follow", "--details", "--tail", tail)

	cmd := exec.CommandContext(ctx, "docker", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open docker logs stdout pipe: %w", err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start docker service logs command: %w", err)
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 256*1024), 256*1024) // 256 KB to handle long TTY lines
	for sc.Scan() {
		line := sc.Text()
		formattedLine, nodeName, taskID := m.formatLogLineWithNode(service.ServiceName, line)
		select {
		case lines <- nodeName + "\x00" + taskID + "\x00" + formattedLine:
		case <-ctx.Done():
			_ = cmd.Wait()
			return nil
		}
	}

	if err := sc.Err(); err != nil {
		return fmt.Errorf("failed reading docker service logs output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return nil
		}
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("docker service logs command failed: %w: %s", err, msg)
		}
		return fmt.Errorf("docker service logs command failed: %w", err)
	}

	return nil
}

// shouldFallbackToRawFromStdCopy returns true when a stdcopy error indicates
// the service uses TTY mode and needs raw log streaming instead.
// NOTE: Error strings are coupled to Docker daemon messages — update if Docker changes them.
func shouldFallbackToRawFromStdCopy(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "tty service logs only supported with --raw") {
		return true
	}
	if strings.Contains(errText, "unrecognized input header") {
		return true
	}
	return false
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
// Output format: formatted line, node name and full task ID for filtering
// Returns: ("service_name.task_id@node_name | actual log message", "node_name", "task_id")
func (m *Model) formatLogLineWithNode(serviceName string, line string) (string, string, string) {
	// Check if line has Docker details prefix
	if !strings.Contains(line, "com.docker.swarm.") {
		return line, "", ""
	}

	// Split on first space to separate details from message
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return line, "", ""
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
	nodeName := m.getNodeHostname(nodeID)
	if nodeName == "" {
		if len(nodeID) >= 12 {
			nodeName = nodeID[:12]
		} else {
			nodeName = nodeID
		}
	}

	// Format task ID (show first 12 chars)
	taskIDShort := taskID
	if len(taskID) > 12 {
		taskIDShort = taskID[:12]
	}

	// Build the formatted prefix with blue color (117 is the light blue we use elsewhere)
	// ANSI escape: \033[38;5;117m for foreground color 117, \033[0m to reset
	prefix := fmt.Sprintf("\033[38;5;117m%s.%s@%s\033[0m", serviceName, taskIDShort, nodeName)

	return fmt.Sprintf("%s | %s", prefix, message), nodeName, taskID
}

// getNodeHostname retrieves the hostname for a node ID from the snapshot
func (m *Model) getNodeHostname(nodeID string) string {
	snapshotOps := m.deps.Snapshot
	snap := snapshotOps.GetSnapshot()
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
