// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"fmt"

	"swarmcli/args"
	"swarmcli/registry"
	helpview "swarmcli/views/help"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Help struct{}

func (Help) Name() string        { return "help" }
func (Help) Description() string { return "Show all available commands" }

func (Help) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Usage:    "[command]",
		Examples: []string{":help", ":help bootstrap"},
	}
}

func (Help) Execute(_ any, a args.Args) tea.Cmd {
	if len(a.Positionals) == 0 {
		return func() tea.Msg {
			return view.NavigateToMsg{
				ViewName: helpview.ViewName,
				Payload:  AllCommandInfos(),
			}
		}
	}

	name := a.Positionals[0]
	target, ok := registry.Get(name)
	if !ok {
		msg := fmt.Sprintf("unknown command: %s", name)
		if s := suggestCommand(name); s != "" {
			msg += fmt.Sprintf(", did you mean :%s?", s)
		}
		return func() tea.Msg { return view.AppErrorMsg{Error: msg} }
	}
	return CommandHelpCmd(target)
}

// CommandHelpCmd returns the tea.Cmd that navigates to the detailed
// help screen for a single command. Shared by `:help <cmd>` and the
// `--help`/`-h` interception path in the dispatcher.
func CommandHelpCmd(cmd registry.Command) tea.Cmd {
	spec, _ := registry.SpecOf(cmd)
	ch := helpview.CommandHelp{
		Title:    ":" + cmd.Name(),
		Detail:   spec.Detail,
		Sections: CommandHelpCategories(cmd),
	}
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: helpview.ViewName,
			Payload:  ch,
		}
	}
}

// CommandHelpCategories renders a command's spec into the detailed help
// view's category model (reused as-is via the help-view factory).
func CommandHelpCategories(cmd registry.Command) []helpview.HelpCategory {
	spec, hasSpec := registry.SpecOf(cmd)

	if !hasSpec || (len(spec.Flags) == 0 && spec.Usage == "" && len(spec.Examples) == 0) {
		return []helpview.HelpCategory{{
			Title: "Usage",
			Items: []helpview.HelpItem{{
				Keys:        ":" + cmd.Name(),
				Description: cmd.Description(),
			}},
		}}
	}

	var cats []helpview.HelpCategory

	usage := ":" + cmd.Name()
	if spec.Usage != "" {
		usage += " " + spec.Usage
	}
	cats = append(cats, helpview.HelpCategory{
		Title: "Usage",
		Items: []helpview.HelpItem{{Keys: usage, Description: cmd.Description()}},
	})

	if len(spec.Flags) > 0 {
		items := make([]helpview.HelpItem, 0, len(spec.Flags))
		for _, f := range spec.Flags {
			keys := "--" + f.Name
			if f.Short != "" {
				keys += ", -" + f.Short
			}
			if f.TakesValue && f.Placeholder != "" {
				keys += " " + f.Placeholder
			}
			items = append(items, helpview.HelpItem{Keys: keys, Description: f.Description})
		}
		cats = append(cats, helpview.HelpCategory{Title: "Flags", Items: items})
	}

	if len(spec.Examples) > 0 {
		items := make([]helpview.HelpItem, 0, len(spec.Examples))
		for _, ex := range spec.Examples {
			items = append(items, helpview.HelpItem{Keys: ex})
		}
		cats = append(cats, helpview.HelpCategory{Title: "Examples", Items: items})
	}

	return cats
}

// suggestCommand returns the closest registered command name to input
// within an edit-distance threshold, or "" if none is close enough.
func suggestCommand(input string) string {
	threshold := len(input) / 3
	if threshold < 2 {
		threshold = 2
	}
	best := ""
	bestDist := threshold + 1
	for _, name := range registry.Suggest("") {
		d := registry.Distance(input, name)
		if d < bestDist || (d == bestDist && name < best) {
			best, bestDist = name, d
		}
	}
	if bestDist > threshold {
		return ""
	}
	return best
}

func AllCommandInfos() []helpview.CommandInfo {
	var cmds []helpview.CommandInfo
	// Go technicality. Need to call `registry` directly.
	// We can't depend on the parent package, as it creates
	// a cycle.
	for _, cmd := range registry.PrimaryCommands() {
		cmds = append(cmds, helpview.CommandInfo{
			Name:        cmd.Name(),
			Description: cmd.Description(),
			Aliases:     cmd.Aliases,
		})
	}
	return cmds
}

var helpCmd = Help{}

func init() {
	registry.Register(helpCmd)
}
