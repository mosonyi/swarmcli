// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview_test

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	chartsview "github.com/Eldara-Tech/swarmcli/views/charts"
	configsview "github.com/Eldara-Tech/swarmcli/views/configs"
	contextsview "github.com/Eldara-Tech/swarmcli/views/contexts"
	helpview "github.com/Eldara-Tech/swarmcli/views/help"
	networksview "github.com/Eldara-Tech/swarmcli/views/networks"
	nodesview "github.com/Eldara-Tech/swarmcli/views/nodes"
	secretsview "github.com/Eldara-Tech/swarmcli/views/secrets"
	servicesview "github.com/Eldara-Tech/swarmcli/views/services"
	stacksview "github.com/Eldara-Tech/swarmcli/views/stacks"
	tasksview "github.com/Eldara-Tech/swarmcli/views/tasks"
	volumesview "github.com/Eldara-Tech/swarmcli/views/volumes"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

// shipped is every cheat sheet in this repo. The layout is only worth as much
// as it is worth for the content that actually reaches an operator, and the
// content is what a table of hand-written fixtures would keep missing: charts
// alone carries a 124-rune description, which is what broke this.
func shipped() map[string][]helpview.HelpCategory {
	return map[string][]helpview.HelpCategory{
		"charts":   chartsview.GetChartsHelpContent(),
		"configs":  configsview.GetConfigsHelpContent(),
		"contexts": contextsview.GetContextsHelpContent(),
		"networks": networksview.GetNetworksHelpContent(),
		"nodes":    nodesview.GetNodesHelpContent(),
		"secrets":  secretsview.GetSecretsHelpContent(),
		"services": servicesview.GetServicesHelpContent(),
		"stacks":   stacksview.GetStacksHelpContent(),
		"tasks":    tasksview.GetTasksHelpContent(),
		"volumes":  volumesview.GetVolumesHelpContent(),
	}
}

// terminalWidths spans a phone-sized split pane to an ultrawide. 102 and 108
// are here by name: the old arithmetic panicked at exactly those widths with
// the charts content, and nothing about them is otherwise special.
var terminalWidths = []int{60, 80, 102, 108, 120, 160, 200, 300}

func render(width, height int, categories []helpview.HelpCategory) string {
	return helpview.NewDetailed(width, height, categories).FrameContent()
}

func TestLayout_StaysInsideTheTerminal(t *testing.T) {
	for name, categories := range shipped() {
		for _, width := range terminalWidths {
			t.Run(fmt.Sprintf("%s/%d", name, width), func(t *testing.T) {
				out := render(width, 40, categories)
				require.True(t, utf8.ValidString(out), "a cut inside a rune leaves invalid UTF-8")
				for i, line := range strings.Split(out, "\n") {
					require.LessOrEqual(t, lipgloss.Width(line), width,
						"line %d overflows the terminal: %q", i, ansi.Strip(line))
				}
			})
		}
	}
}

// TestLayout_KeepsEveryWord is the assertion the screenshot on #593 was of. At
// width 60 every content lays out in a single column, so a description that
// survives can be found in the output with its wrapping whitespace removed.
func TestLayout_KeepsEveryWord(t *testing.T) {
	for name, categories := range shipped() {
		t.Run(name, func(t *testing.T) {
			out := squash(render(60, 400, categories))
			for _, cat := range categories {
				for _, item := range cat.Items {
					require.Contains(t, out, squash(item.Description),
						"the %s help cuts %q", name, item.Description)
					require.Contains(t, out, squash(item.Keys))
				}
			}
		})
	}
}

// TestLayout_UsesTheWidthItIsGiven guards the other direction: wrapping every
// description would be readable and waste an ultrawide terminal.
func TestLayout_UsesTheWidthItIsGiven(t *testing.T) {
	categories := chartsview.GetChartsHelpContent()

	// The rendered page is always the frame's height; what a wider terminal
	// buys is fewer rows carrying content, because the categories sit beside
	// each other rather than under.
	narrow := filledRows(render(60, 400, categories))
	wide := filledRows(render(240, 400, categories))

	require.Greater(t, narrow, wide,
		"a wider terminal should take columns, not the same single column")
	require.Positive(t, wide)
}

// TestLayout_KeyThatIsReallyAPhrase: some categories document values rather
// than keystrokes. The key column is capped at half the block so one long
// entry cannot leave every description in its category a sliver.
func TestLayout_KeyThatIsReallyAPhrase(t *testing.T) {
	categories := []helpview.HelpCategory{{
		Title: "When a pane says something instead of drawing",
		Items: []helpview.HelpItem{
			{Keys: "not reported by this host", Description: "No sample in the window carried this metric"},
			{Keys: "<w>", Description: "Cycle the window"},
		},
	}}

	out := render(70, 40, categories)
	squashed := squash(out)
	for _, item := range categories[0].Items {
		require.Contains(t, squashed, squash(item.Description))
		require.Contains(t, squashed, squash(item.Keys))
	}
	for _, line := range strings.Split(out, "\n") {
		require.LessOrEqual(t, lipgloss.Width(line), 70)
	}
}

// TestLayout_WrapsUnderTheDescription guards the shape of a wrapped item, not
// merely that it fits: the column is padded to its width at the end, so text
// that was never wrapped would be folded there anyway — flush against the left
// edge, under the key rather than under the sentence it continues.
func TestLayout_WrapsUnderTheDescription(t *testing.T) {
	const firstWord = "aardvark"
	categories := []helpview.HelpCategory{{
		Title: "General",
		Items: []helpview.HelpItem{{
			Keys:        "<x>",
			Description: firstWord + " sentence long enough that it has to wrap several times over at this narrow width",
		}},
	}}

	var body []string
	for _, line := range strings.Split(ansi.Strip(render(40, 40, categories)), "\n") {
		if strings.TrimSpace(line) != "" {
			body = append(body, strings.TrimRight(line, " "))
		}
	}
	require.GreaterOrEqual(t, len(body), 4, "the description must have wrapped more than once")

	indent := strings.Index(body[1], firstWord) // body[0] is the category title
	require.Greater(t, indent, 0, "the description starts after the key")
	for i, line := range body[2:] {
		require.True(t, strings.HasPrefix(line, strings.Repeat(" ", indent)),
			"continuation line %d starts under the key instead of the description: %q", i, line)
		require.NotEqual(t, byte(' '), line[indent],
			"continuation line %d does not start at the description column: %q", i, line)
	}
}

// TestLayout_ScrollsRatherThanLosingTheTail: wrapping makes screens taller, and
// a frame that cannot show all of it must be able to reach the rest.
func TestLayout_ScrollsRatherThanLosingTheTail(t *testing.T) {
	categories := chartsview.GetChartsHelpContent()
	m := helpview.NewDetailed(60, 12, categories)

	first := m.FrameContent()
	require.LessOrEqual(t, len(strings.Split(first, "\n")), 12)
	require.Contains(t, m.FrameFooter(), "scroll", "the footer must offer the keys that reach the rest")

	m.Viewable.GotoBottom()
	last := m.FrameContent()
	require.NotEqual(t, squash(first), squash(last), "the page must move")

	lastCategory := categories[len(categories)-1]
	lastItem := lastCategory.Items[len(lastCategory.Items)-1]
	require.Contains(t, squash(last), squash(lastItem.Description),
		"the end of the sheet must be reachable")
}

func TestLayout_ShortSheetOffersNoScrollKeys(t *testing.T) {
	m := helpview.NewDetailed(200, 40, tasksview.GetTasksHelpContent())
	m.FrameContent()
	require.NotContains(t, m.FrameFooter(), "scroll")
}

// filledRows counts the rows that carry something, as opposed to the blank
// ones the viewport pads its page out with.
func filledRows(s string) int {
	n := 0
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(ansi.Strip(line)) != "" {
			n++
		}
	}
	return n
}

// squash strips styling and every space and newline, so a string that was
// wrapped across lines and columns still matches the sentence it came from,
// while a truncated one does not.
func squash(s string) string {
	return strings.Join(strings.Fields(ansi.Strip(s)), "")
}
