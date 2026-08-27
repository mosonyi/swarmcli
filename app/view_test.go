// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"github.com/Eldara-Tech/swarmcli/v2/views/commandinput"
	"github.com/Eldara-Tech/swarmcli/v2/views/confirmdialog"
	inspectview "github.com/Eldara-Tech/swarmcli/v2/views/inspect"
	"github.com/Eldara-Tech/swarmcli/v2/views/searchinput"
	systeminfoview "github.com/Eldara-Tech/swarmcli/v2/views/systeminfo"
	"github.com/Eldara-Tech/swarmcli/v2/views/unlockdialog"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"
	"github.com/Eldara-Tech/swarmcli/v2/views/viewstack"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// frameStubView sizes its content the way the real views do: from the frame
// height the app hands it, minus the frame's own rows and its header/footer.
// It always has more content than fits, so any row the layout fails to claim
// shows up as a shortfall rather than as content running out.
type frameStubView struct {
	stubView
	frameHeight int
	header      string
	footer      string
}

func (v *frameStubView) Update(msg tea.Msg) tea.Cmd {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		v.frameHeight = ws.Height
	}
	return nil
}

func (v *frameStubView) FrameTitle() string  { return "Stub" }
func (v *frameStubView) FrameHeader() string { return v.header }
func (v *frameStubView) FrameFooter() string { return v.footer }

func (v *frameStubView) FrameContent() string {
	return ui.TrimOrPadContentToLines(strings.Repeat("row\n", 500),
		ui.ContentRows(v.frameHeight, ui.FramedChromeRows, v.header, v.footer))
}

// stubClusterInfo is the least that lets systeminfoview render.
type stubClusterInfo struct{}

func (stubClusterInfo) GetCurrentContext() (string, error)    { return "default", nil }
func (stubClusterInfo) GetContainerCount() (int, error)       { return 0, nil }
func (stubClusterInfo) GetServiceCount() (int, error)         { return 0, nil }
func (stubClusterInfo) GetSwarmCPUCapacity() (float64, error) { return 0, nil }
func (stubClusterInfo) GetSwarmMemCapacity() (int64, error)   { return 0, nil }
func (stubClusterInfo) GetSwarmCPUUsage() (string, error)     { return "0%", nil }
func (stubClusterInfo) GetSwarmMemUsage() (string, error)     { return "0%", nil }
func (stubClusterInfo) GetSwarmResourceUsage() (string, string, error) {
	return "0%", "0%", nil
}
func (stubClusterInfo) GetDockerVersion() (string, error) { return "27.0.0", nil }

// newLayoutTestModel builds a model complete enough to run View().
func newLayoutTestModel(cv view.View) *Model {
	deps := docker.Deps{ClusterInfo: stubClusterInfo{}}
	return &Model{
		deps:               deps,
		viewport:           viewport.New(0, 0),
		currentView:        cv,
		viewStack:          viewstack.Stack{},
		commandInput:       commandinput.New(),
		searchInput:        searchinput.New(),
		errorDialog:        confirmdialog.New(0, 0),
		unlockDialog:       unlockdialog.New(0, 0),
		updateDialog:       confirmdialog.New(0, 0),
		contextDriftDialog: confirmdialog.New(0, 0),
		systemInfo:         systeminfoview.New(deps, "test", "ce"),
	}
}

// TestViewFillsTerminalHeight is the guard for #560: fullscreen used to hold
// back rows for frame borders it never draws, leaving a dead band at the
// bottom of the terminal. Every layout must claim every row it is given.
func TestViewFillsTerminalHeight(t *testing.T) {
	const width, height = 100, 40

	cases := []struct {
		name           string
		fullscreen     bool
		header, footer string
		openBar        func(*Model)
	}{
		{name: "normal", header: "hdr", footer: "ftr"},
		{name: "fullscreen", fullscreen: true, header: "hdr", footer: "ftr"},
		{name: "fullscreen without header or footer", fullscreen: true},
		{name: "fullscreen with a two-line footer", fullscreen: true, header: "hdr", footer: "a\nb"},
		{
			name: "normal with the command bar open", header: "hdr", footer: "ftr",
			openBar: func(m *Model) { m.commandInput.Show() },
		},
		{
			name: "fullscreen with the command bar open", fullscreen: true, header: "hdr", footer: "ftr",
			openBar: func(m *Model) { m.commandInput.Show() },
		},
		{
			name: "fullscreen with the search bar open", fullscreen: true, header: "hdr", footer: "ftr",
			openBar: func(m *Model) { m.searchInput.Show() },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newLayoutTestModel(&frameStubView{header: tc.header, footer: tc.footer})
			m.fullscreen = tc.fullscreen
			if tc.openBar != nil {
				tc.openBar(m)
			}
			m.updateForResize(tea.WindowSizeMsg{Width: width, Height: height})

			// What the chrome costs, stated independently of the code under
			// test: fullscreen spends one row on its centered title, the normal
			// layout spends six on the help bar, one on the stack bar and two on
			// the frame's borders. Either way an open input bar costs three.
			chrome := 1
			if !tc.fullscreen {
				chrome = 6 + 1 + 2
			}
			if tc.openBar != nil {
				chrome += 3
			}
			wantContent := height - chrome - rowsIn(tc.header) - rowsIn(tc.footer)

			out := m.View()
			require.Equal(t, height, lipgloss.Height(out), "rendered rows")
			// The view has more content than fits, so every content row it was
			// given should carry content. Blank rows mean the layout reserved
			// space it never used — the dead band from #560.
			require.Equal(t, wantContent, countRowsWith(out, "row"), "rows of content drawn")
			for i, line := range strings.Split(out, "\n") {
				require.LessOrEqual(t, lipgloss.Width(line), width, "row %d overflows the terminal width", i)
			}
		})
	}
}

// rowsIn is the number of rows a header or footer occupies; "" occupies none.
func rowsIn(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func countRowsWith(out, marker string) int {
	n := 0
	for _, row := range strings.Split(out, "\n") {
		if strings.Contains(row, marker) {
			n++
		}
	}
	return n
}

// TestFullscreenFillsTheScreenWithARealViewport — the stub above rebuilds its
// content for whatever height it is handed, so it cannot show what a real view
// does: scroll a viewport that carries its offset across the resize. Pressing
// "f" grew the viewport, which went on drawing the lines it drew before and
// left the rows the toggle gained blank, until a keypress pulled the offset
// back into range.
func TestFullscreenFillsTheScreenWithARealViewport(t *testing.T) {
	const width, height = 100, 40

	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, fmt.Sprintf("line-%03d", i))
	}
	inspect := inspectview.New(width, height, inspectview.FormatRaw)
	inspect.SetContent(strings.Join(lines, "\n"))

	m := newLayoutTestModel(inspect)
	m.updateForResize(tea.WindowSizeMsg{Width: width, Height: height})
	for i := 0; i < 20; i++ { // page down to the end of the document
		m.handleKey(tea.KeyMsg{Type: tea.KeyPgDown})
	}

	m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})

	out := m.View()
	require.Equal(t, height, lipgloss.Height(out), "rendered rows")
	// 40 rows less the fullscreen title line and the view's own header.
	require.Equal(t, 38, countRowsWith(out, "line-"), "rows of content drawn")
	rows := strings.Split(out, "\n")
	require.Contains(t, rows[len(rows)-1], "line-499", "the last line should sit on the last row")
}

// TestFullscreenKeepsTheViewFooter — fullscreen used to drop the footer while
// the view still reserved its rows, so the row counter and the deploy progress
// line vanished and their space was wasted twice over.
func TestFullscreenKeepsTheViewFooter(t *testing.T) {
	m := newLayoutTestModel(&frameStubView{header: "hdr", footer: "footer-marker"})
	m.fullscreen = true
	m.updateForResize(tea.WindowSizeMsg{Width: 100, Height: 40})

	rows := strings.Split(m.View(), "\n")
	require.Contains(t, rows[len(rows)-1], "footer-marker")
}

// TestInputBarStaysVisibleInFullscreen — ":" and "/" used to accept keystrokes
// in fullscreen while View() returned before drawing either bar, so you typed
// blind.
func TestInputBarStaysVisibleInFullscreen(t *testing.T) {
	for _, tc := range []struct {
		name string
		open func(*Model)
	}{
		{"command bar", func(m *Model) { m.commandInput.Show() }},
		{"search bar", func(m *Model) { m.searchInput.Show() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newLayoutTestModel(&frameStubView{header: "hdr", footer: "ftr"})
			m.fullscreen = true
			tc.open(m)
			m.updateForResize(tea.WindowSizeMsg{Width: 100, Height: 40})

			rows := strings.Split(m.View(), "\n")
			require.Equal(t, inputBarHeight, indexOfStubTitle(t, rows),
				"the input bar should occupy the rows above the view")
		})
	}
}

// indexOfStubTitle returns the row the stub view's title line lands on.
func indexOfStubTitle(t *testing.T, rows []string) int {
	t.Helper()
	for i, row := range rows {
		if strings.Contains(row, "Stub") {
			return i
		}
	}
	t.Fatal("the view's title line is missing from the output")
	return -1
}

// TestNavigatingInFullscreenKeepsTheFullscreenBudget — every resize outside the
// window-size handler used to hard-code normal mode, so drilling into a view
// while fullscreen sized it for the help and stack bars that fullscreen does
// not draw, wasting even more of the screen than the toggle itself did.
func TestNavigatingInFullscreenKeepsTheFullscreenBudget(t *testing.T) {
	const name = "frame-stub"
	view.RegisterView(name, func(docker.Deps, int, int, any) (view.View, tea.Cmd) {
		return &frameStubView{header: "hdr", footer: "ftr"}, nil
	})

	m := newLayoutTestModel(&frameStubView{header: "hdr", footer: "ftr"})
	m.fullscreen = true
	m.updateForResize(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.switchToView(name, nil)

	out := m.View()
	require.Equal(t, 40, lipgloss.Height(out))
	// 40 rows less the title, the header and the footer.
	require.Equal(t, 37, countRowsWith(out, "row"))
}

// TestHandleViewResizeFullscreenBudget pins the contract between the resize
// math and the fullscreen frame: the height a view is handed must leave it,
// after the deduction every view makes for the bordered frame, exactly the
// rows the fullscreen frame will draw.
func TestHandleViewResizeFullscreenBudget(t *testing.T) {
	const terminalHeight = 40
	const header, footer = "hdr", "ftr"

	v := &frameStubView{header: header, footer: footer}
	handleViewResize(v, 100, terminalHeight, true)

	require.Equal(t,
		ui.ContentRows(terminalHeight, ui.FullscreenChromeRows, header, footer),
		ui.ContentRows(v.frameHeight, ui.FramedChromeRows, header, footer))
}
