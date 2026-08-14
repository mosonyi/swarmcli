// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"fmt"
	"testing"

	"github.com/Eldara-Tech/swarmcli/args"
	contextsview "github.com/Eldara-Tech/swarmcli/views/contexts"
	"github.com/Eldara-Tech/swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

// stubUpdate replaces the Docker seams for one test, recording the call.
func stubUpdate(t *testing.T, active string, err error) *[]string {
	t.Helper()
	calls := &[]string{}
	origUpdate, origActive := updateContextEndpoint, activeContextName
	t.Cleanup(func() { updateContextEndpoint, activeContextName = origUpdate, origActive })

	updateContextEndpoint = func(name, description, host string) error {
		*calls = append(*calls, fmt.Sprintf("%s|%s|%s", name, description, host))
		return err
	}
	activeContextName = func() (string, error) { return active, nil }
	return calls
}

// updateArgs is what api.ParseInput hands Execute for
// ':context update <name> --host <endpoint>'.
func updateArgs(positionals []string, host string) args.Args {
	a := args.Args{Positionals: positionals, Flags: map[string]string{}}
	if host != "" {
		a.Flags["host"] = host
	}
	return a
}

func TestContexts_NoArgs_Navigates(t *testing.T) {
	msg := contextsCmd.Execute(nil, args.Args{})()
	require.IsType(t, view.NavigateToMsg{}, msg)
	require.Equal(t, contextsview.ViewName, msg.(view.NavigateToMsg).ViewName)
}

func TestContexts_Update_CallsDocker(t *testing.T) {
	calls := stubUpdate(t, "other", nil)
	msg := contextsCmd.Execute(nil, updateArgs([]string{"update", "prod"}, "tcp://10.0.0.7:2376"))()
	require.Equal(t, []string{"prod||tcp://10.0.0.7:2376"}, *calls)
	require.IsType(t, view.AppInfoMsg{}, msg)
	require.Contains(t, msg.(view.AppInfoMsg).Message, "tcp://10.0.0.7:2376")
}

// The cached client was built for the endpoint that just moved, so the app has
// to rebuild it — the same path a context switch takes.
func TestContexts_Update_ActiveContext_Reconnects(t *testing.T) {
	stubUpdate(t, "prod", nil)
	msg := contextsCmd.Execute(nil, updateArgs([]string{"update", "prod"}, "tcp://10.0.0.7:2376"))()
	require.IsType(t, contextsview.ContextChangedNotification{}, msg)
}

func TestContexts_Update_ReportsFailure(t *testing.T) {
	stubUpdate(t, "prod", fmt.Errorf("context 'prod' does not exist"))
	msg := contextsCmd.Execute(nil, updateArgs([]string{"update", "prod"}, "tcp://10.0.0.7:2376"))()
	require.IsType(t, view.AppErrorMsg{}, msg)
	require.Contains(t, msg.(view.AppErrorMsg).Error, "does not exist")
}

func TestContexts_Update_ArgumentErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		positionals []string
		host        string
		want        string
	}{
		"unknown subcommand": {[]string{"delete", "prod"}, "tcp://h:2376", "unknown subcommand 'delete'"},
		"no name":            {[]string{"update"}, "tcp://h:2376", "one context name is required"},
		"extra positional":   {[]string{"update", "prod", "extra"}, "tcp://h:2376", "one context name is required"},
		"no host":            {[]string{"update", "prod"}, "", "--host needs an endpoint"},
		"blank host":         {[]string{"update", "prod"}, "   ", "--host needs an endpoint"},
	} {
		t.Run(name, func(t *testing.T) {
			calls := stubUpdate(t, "other", nil)
			msg := contextsCmd.Execute(nil, updateArgs(tc.positionals, tc.host))()
			require.IsType(t, view.AppErrorMsg{}, msg)
			require.Contains(t, msg.(view.AppErrorMsg).Error, tc.want)
			require.Empty(t, *calls, "Docker must not be called for a rejected invocation")
		})
	}
}
