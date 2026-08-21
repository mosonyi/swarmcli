// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"fmt"
	"strings"

	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	contextsview "github.com/Eldara-Tech/swarmcli/v2/views/contexts"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// updateSubcommand turns the command into an in-place edit instead of a
// navigation to the contexts view.
const updateSubcommand = "update"

// Seams, so the subcommand is testable without a Docker CLI.
var (
	updateContextEndpoint = docker.UpdateContextEndpoint
	activeContextName     = docker.GetContextFromEnv
)

type Contexts struct{}

func (Contexts) Name() string        { return "contexts" }
func (Contexts) Description() string { return "List and switch Docker contexts" }

func (Contexts) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Usage: "[update <name> --host <endpoint>]",
		Detail: "Opens the Docker context list, where you can switch the " +
			"active context (which reloads the cluster) and create, " +
			"inspect, edit, delete, import or export contexts.\n\n" +
			"With 'update <name>', repoints that context at another " +
			"endpoint in place, keeping any TLS material it already has. " +
			"Everything else about the context is left as it is; press 'e' " +
			"in the context list to edit its description.",
		Flags: []registry.FlagSpec{
			{
				Name:        "host",
				TakesValue:  true,
				Placeholder: "<endpoint>",
				Description: "Docker endpoint to point the context at, e.g. tcp://10.0.0.7:2376",
			},
		},
		Examples: []string{
			":contexts",
			":context update prod --host tcp://10.0.0.7:2376",
		},
	}
}

func (Contexts) Execute(ctx any, args args.Args) tea.Cmd {
	if len(args.Positionals) > 0 {
		return updateContextCmd(args)
	}
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: contextsview.ViewName,
			Payload:  nil,
		}
	}
}

// updateContextCmd runs ':context update <name> --host <endpoint>'. Docker
// replaces a context's whole endpoint on update, so the docker package's
// update is what carries the stored TLS material across a host change.
func updateContextCmd(a args.Args) tea.Cmd {
	return func() tea.Msg {
		name, err := updateTarget(a)
		if err != nil {
			return view.AppErrorMsg{Error: err.Error()}
		}

		host := strings.TrimSpace(a.Get("host"))
		if err := updateContextEndpoint(name, "", host); err != nil {
			return view.AppErrorMsg{Error: fmt.Sprintf("Failed to update context '%s': %v", name, err)}
		}

		// A moved endpoint leaves the cached client and snapshot describing a
		// daemon we are no longer pointing at, so the active context has to be
		// reconnected to. The contexts view takes the same path after an edit.
		if active, err := activeContextName(); err == nil && active == name {
			return contextsview.ContextChangedNotification{}
		}
		return view.AppInfoMsg{Message: fmt.Sprintf("Updated context '%s' — host is now %s", name, host)}
	}
}

// updateTarget validates the subcommand's arguments and returns the context to
// update.
func updateTarget(a args.Args) (string, error) {
	if a.Positionals[0] != updateSubcommand {
		return "", fmt.Errorf("unknown subcommand '%s' — usage: :contexts [update <name> --host <endpoint>]", a.Positionals[0])
	}
	if len(a.Positionals) != 2 {
		return "", fmt.Errorf("one context name is required — usage: :context update <name> --host <endpoint>")
	}
	if strings.TrimSpace(a.Get("host")) == "" {
		return "", fmt.Errorf("--host needs an endpoint, e.g. --host tcp://10.0.0.7:2376")
	}
	return a.Positionals[1], nil
}

var contextsCmd = Contexts{}

func init() {
	registry.Register(contextsCmd)
	// Register aliases
	registry.Register(aliasCommand{name: "context", target: contextsCmd})
	registry.Register(aliasCommand{name: "ctx", target: contextsCmd})
}
