// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"sort"
	"strings"

	"github.com/Eldara-Tech/swarmcli/registry"
)

// chartsCmd describes one `swarmcli charts` subcommand: what it is called, what
// it takes, and what it does. The table below is the single source of truth
// behind dispatch, the usage text and the generated command blocks in
// README.md and charts/README.md — three places that each used to carry their
// own hand-maintained copy of the same list, and drifted.
//
// Adding a subcommand means adding a row here. Nothing else has a list to
// update.
type chartsCmd struct {
	// Name is the word after `swarmcli charts`, and the key dispatch looks up.
	Name string
	// Aliases are alternative spellings dispatch accepts. They are reported in
	// the rendered summary rather than given lines of their own.
	Aliases []string
	// Group is the heading this command is listed under. Rendering follows
	// chartGroups order, not declaration order.
	Group string
	// Usage is one or more display lines. A command with sub-verbs (`repo add`,
	// `show values`) lists each of them here: they are one dispatch target but
	// several things an operator can run.
	Usage []cmdUsage
	// Run is the handler. It receives the arguments after the command name.
	Run func([]string) int
}

// cmdUsage is one rendered line: the arguments as an operator types them, and
// what that invocation does.
type cmdUsage struct {
	Args    string
	Summary string
}

// chartGroups is the order the groups appear in, which is roughly the order an
// operator meets them: configure a repository, find a chart, write one, deploy
// it, then automate the deployment.
var chartGroups = []string{"Repository", "Discovery", "Authoring", "Releases", "GitOps"}

// chartsCommands is the command set. Ordering within a group is the order rows
// appear here.
var chartsCommands = []chartsCmd{
	{
		Name:  "repo",
		Group: "Repository",
		Usage: []cmdUsage{
			{"add <name> <url>", "Add a chart repository and download its index"},
			{"list", "List configured repositories"},
			{"update [name]", "Refresh repository indexes (all, or one)"},
			{"remove <name>", "Remove a repository"},
		},
		Run: chartsRepo,
	},
	{
		Name:  "search",
		Group: "Discovery",
		Usage: []cmdUsage{{"[keyword]", "Search charts across repositories"}},
		Run:   chartsSearch,
	},
	{
		Name:  "show",
		Group: "Discovery",
		Usage: []cmdUsage{
			{"chart <repo/chart>", "Show chart metadata"},
			{"values <repo/chart>", "Show default values.yaml"},
			{"schema <repo/chart>", "Show values.schema.json"},
		},
		Run: chartsShow,
	},
	{
		Name:  "lint",
		Group: "Authoring",
		Usage: []cmdUsage{{"<chart>", "Check a chart without deploying it"}},
		Run:   chartsLint,
	},
	{
		Name:  "template",
		Group: "Releases",
		Usage: []cmdUsage{{"<release> <chart>", "Render manifest to stdout (no deploy)"}},
		Run:   chartsTemplate,
	},
	{
		Name:  "install",
		Group: "Releases",
		Usage: []cmdUsage{{"<release> <chart>", "Install a chart as a release"}},
		Run:   chartsInstall,
	},
	{
		Name:  "upgrade",
		Group: "Releases",
		Usage: []cmdUsage{{"<release> <chart>", "Upgrade a release to a new revision"}},
		Run:   chartsUpgrade,
	},
	{
		Name:  "uninstall",
		Group: "Releases",
		Usage: []cmdUsage{{"<release>", "Remove a release (keeps volumes)"}},
		Run:   chartsUninstall,
	},
	{
		Name:  "rollback",
		Group: "Releases",
		Usage: []cmdUsage{{"<release> <rev>", "Re-deploy the contents of a past revision"}},
		Run:   chartsRollback,
	},
	{
		Name:  "history",
		Group: "Releases",
		Usage: []cmdUsage{{"<release>", "Show a release's revision history"}},
		Run:   chartsHistory,
	},
	{
		Name:  "prune",
		Group: "Releases",
		Usage: []cmdUsage{{"[release]", "Delete old revisions beyond --history-max"}},
		Run:   chartsPrune,
	},
	{
		Name:  "get",
		Group: "Releases",
		Usage: []cmdUsage{{"values|manifest <release>", "Show stored values or rendered manifest"}},
		Run:   chartsGet,
	},
	{
		Name:  "diff",
		Group: "Releases",
		Usage: []cmdUsage{{"upgrade <release> <chart>", "Preview manifest changes before upgrading"}},
		Run:   chartsDiff,
	},
	{
		Name:    "list",
		Aliases: []string{"ls"},
		Group:   "Releases",
		Usage:   []cmdUsage{{"", "List releases"}},
		Run:     chartsList,
	},
	{
		Name:  "status",
		Group: "Releases",
		Usage: []cmdUsage{{"<release>", "Show release status and services"}},
		Run:   chartsStatus,
	},
	{
		Name:  "apply",
		Group: "GitOps",
		Usage: []cmdUsage{{"-f <file>", "Converge the swarm to a declarative release file"}},
		Run:   chartsApply,
	},
	{
		Name:  "outdated",
		Group: "GitOps",
		Usage: []cmdUsage{{"", "Show releases with a newer chart version available"}},
		Run:   chartsOutdated,
	},
}

// lookupCommand resolves a name or alias to its row.
func lookupCommand(name string) (chartsCmd, bool) {
	for _, c := range chartsCommands {
		if c.Name == name {
			return c, true
		}
		for _, a := range c.Aliases {
			if a == name {
				return c, true
			}
		}
	}
	return chartsCmd{}, false
}

// commandNames lists every name and alias, sorted so a suggestion breaks ties
// deterministically.
func commandNames() []string {
	var names []string
	for _, c := range chartsCommands {
		names = append(names, c.Name)
		names = append(names, c.Aliases...)
	}
	sort.Strings(names)
	return names
}

// suggestCommand returns the closest command name to input within an edit
// distance threshold, or "" if nothing is close enough. Same shape as the
// TUI's unknown-flag suggestion (commands/api.suggestFlag), including the
// threshold, so a typo is answered the same way on both surfaces.
func suggestCommand(input string, names []string) string {
	threshold := len(input) / 3
	if threshold < 2 {
		threshold = 2
	}
	best := ""
	bestDist := threshold + 1
	for _, n := range names {
		if d := registry.Distance(input, n); d < bestDist {
			best, bestDist = n, d
		}
	}
	if bestDist > threshold {
		return ""
	}
	return best
}

// renderCommands renders the grouped command list. prefix is what each line
// starts with — empty for the usage text, "swarmcli charts " for the README
// blocks, which show commands as an operator would type them.
func renderCommands(prefix string) string {
	type line struct{ left, summary string }

	// One column width across every group, so the block reads as one table
	// rather than a stack of independently aligned ones.
	var lines []line
	width := 0
	for _, g := range chartGroups {
		for _, c := range chartsCommands {
			if c.Group != g {
				continue
			}
			for i, u := range c.Usage {
				left := strings.TrimRight(prefix+c.Name+" "+u.Args, " ")
				summary := u.Summary
				// The alias is reported on the command's first line rather than
				// given a line of its own: it is the same command, and a second
				// entry would read as a second thing to learn.
				if i == 0 && len(c.Aliases) > 0 {
					summary += " (alias: " + strings.Join(c.Aliases, ", ") + ")"
				}
				lines = append(lines, line{left, summary})
				if len(left) > width {
					width = len(left)
				}
			}
		}
	}

	var b strings.Builder
	i := 0
	for _, g := range chartGroups {
		count := 0
		for _, c := range chartsCommands {
			if c.Group == g {
				count += len(c.Usage)
			}
		}
		if count == 0 {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString(g + ":\n")
		for ; count > 0; count-- {
			l := lines[i]
			i++
			b.WriteString("  " + l.left + strings.Repeat(" ", width-len(l.left)+2) + l.summary + "\n")
		}
	}
	return b.String()
}

// chartsUsage is the full `swarmcli charts --help` text: the generated command
// list, then the prose that explains the parts a list cannot.
func chartsUsage() string {
	return "Usage: swarmcli charts <command> [options]\n\n" + renderCommands("") + chartsUsageProse
}
