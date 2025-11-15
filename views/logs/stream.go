package logsview

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
				Details:    false,
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
					lines <- sc.Text() // sc.Text() does not include newline
				}
			}

			wg.Add(2)
			go pushScanner(stdoutR)
			go pushScanner(stderrR)

			// wait for scanners + stdcopy to finish
			wg.Wait()
			scWG.Wait()

			if scErr != nil {
				l().With("service", service.ServiceID).Warnf("stdcopy finished with error: %v", scErr)
			}

			// done: channels will be closed by defer above
			return
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
