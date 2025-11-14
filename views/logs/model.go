package logsview

import (
	"bufio"
	"io"
	"os/exec"
	"swarmcli/docker"
	"swarmcli/views/helpbar"
	"sync"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport      viewport.Model
	Visible       bool
	mode          string // "normal", "search"
	searchTerm    string
	searchIndex   int
	searchMatches []int // line indexes of matches
	lines         []string
	ready         bool

	// streaming channels (set when stream is started)
	linesChan chan string
	errChan   chan error

	// internal mutex to protect lines slice when appended from goroutine
	mu sync.Mutex
}

// New creates a new logs view model
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
		mode:     "normal",
		lines:    make([]string, 0),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.mode == "search" {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "confirm"},
			{Key: "esc", Desc: "cancel"},
			{Key: "n/N", Desc: "next/prev"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "n/N", Desc: "next/prev"},
		{Key: "q", Desc: "close"},
	}
}

// Load starts streaming logs for the given service. It returns a command that
// initializes the background streamer and sends back an InitStreamMsg.
func Load(service docker.ServiceEntry) tea.Cmd {
	return func() tea.Msg {
		lines := make(chan string, 128)
		errs := make(chan error, 1)

		// start the streaming goroutine
		go func() {
			// docker service logs --no-trunc <id>
			cmd := exec.Command("docker", "service", "logs", "--no-trunc", service.ServiceID)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				errs <- err
				close(lines)
				close(errs)
				return
			}
			stderr, err := cmd.StderrPipe()
			if err != nil {
				errs <- err
				close(lines)
				close(errs)
				return
			}

			if err := cmd.Start(); err != nil {
				errs <- err
				close(lines)
				close(errs)
				return
			}

			// scan stdout and stderr concurrently, push lines into lines chan
			var wg sync.WaitGroup
			pushScanner := func(r io.Reader) {
				defer wg.Done()
				sc := bufio.NewScanner(r)
				for sc.Scan() {
					// preserve newline
					lines <- sc.Text() + "\n"
				}
				// ignore scan error here; we'll check after Wait
			}

			wg.Add(2)
			go pushScanner(stdout)
			go pushScanner(stderr)

			// wait for scanners
			wg.Wait()

			// wait for process exit
			if err := cmd.Wait(); err != nil {
				// If docker returns non-zero (could be expected), still close lines but report error
				errs <- err
			}
			close(lines)
			close(errs)
		}()

		// return init message which carries the channels
		return InitStreamMsg{
			Lines: lines,
			Errs:  errs,
		}
	}
}
