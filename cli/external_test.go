// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// withExternals installs a registry for one test and restores the real one,
// which is empty in this repository's own build and must stay that way.
func withExternals(t *testing.T) {
	t.Helper()
	orig := externalCommands
	externalCommands = map[string]externalCommand{}
	t.Cleanup(func() { externalCommands = orig })
}

// captureOut redirects the package writers for one test.
func captureOut(t *testing.T) (*bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var o, e bytes.Buffer
	origOut, origErr := stdout, stderr
	stdout, stderr = &o, &e
	t.Cleanup(func() { stdout, stderr = origOut, origErr })
	return &o, &e
}

func TestDispatch_RoutesToARegisteredCommand(t *testing.T) {
	withExternals(t)
	captureOut(t)

	var got []string
	RegisterCommand("license", "Show or renew the licence", func(args []string) int {
		got = args
		return 3
	})

	code := Dispatch([]string{"license", "status", "--json"}, "v1.14.0")
	require.Equal(t, 3, code, "the verb owns its exit code; this package does not translate it")
	require.Equal(t, []string{"status", "--json"}, got, "everything after the verb, unparsed")
}

// The seam extends the vocabulary and cannot change it. A build that could
// re-point `version` or `charts` would be a build whose behaviour no longer
// follows from the source somebody is reading.
func TestRegisterCommand_RefusesToShadowOrDuplicate(t *testing.T) {
	withExternals(t)

	for _, name := range []string{"charts", "version", "--version", "-v", "help", "--help", "-h"} {
		require.PanicsWithValue(t,
			"cli: RegisterCommand("+name+") would shadow a built-in command",
			func() { RegisterCommand(name, "hijacked", func([]string) int { return 0 }) },
			"a built-in must not be shadowable: %q", name)
	}

	RegisterCommand("license", "first", func([]string) int { return 0 })
	require.PanicsWithValue(t, "cli: RegisterCommand(license) registered twice",
		func() { RegisterCommand("license", "second", func([]string) int { return 0 }) })

	require.Panics(t, func() { RegisterCommand("", "nameless", func([]string) int { return 0 }) })
	require.Panics(t, func() { RegisterCommand("noop", "no run", nil) })
}

func TestDispatch_BuiltinsAreUnaffectedByRegistration(t *testing.T) {
	withExternals(t)
	o, _ := captureOut(t)

	RegisterCommand("license", "Show or renew the licence", func([]string) int {
		t.Fatal("a registered verb must never be reached for a built-in")
		return 0
	})

	require.Equal(t, 0, Dispatch([]string{"version"}, "v1.14.0"))
	require.Contains(t, o.String(), "v1.14.0")
}

// Help lists what this binary can actually do — which is why the seam is a
// registration rather than a name this package knows: with nothing
// registered, the OSS build's help is exactly what it was.
func TestTopUsage_ListsRegisteredCommandsOnly(t *testing.T) {
	withExternals(t)

	require.Equal(t, topUsageHead+topUsageTail, topUsage(),
		"a build that registered nothing advertises nothing")

	RegisterCommand("license", "Show or renew the licence", func([]string) int { return 0 })
	RegisterCommand("bootstrap", "Deploy the infrastructure stack", func([]string) int { return 0 })

	usage := topUsage()
	require.Contains(t, usage, "  bootstrap   Deploy the infrastructure stack")
	require.Contains(t, usage, "  license     Show or renew the licence")
	require.Less(t, strings.Index(usage, "bootstrap"), strings.Index(usage, "license"),
		"stable order, not map iteration")
	require.Less(t, strings.Index(usage, "charts"), strings.Index(usage, "bootstrap"),
		"built-ins first: they are what this binary is")
	require.True(t, strings.HasSuffix(usage, topUsageTail))
}

func TestDispatch_UnknownCommandListsRegisteredOnes(t *testing.T) {
	withExternals(t)
	_, e := captureOut(t)

	RegisterCommand("license", "Show or renew the licence", func([]string) int { return 0 })

	code := Dispatch([]string{"licence"}, "v1.14.0")
	require.Equal(t, 2, code, "usage errors exit 2")
	require.Contains(t, e.String(), "unknown command \"licence\"")
	require.Contains(t, e.String(), "license", "the suggestion an operator needs is the list itself")
}

// The built-in list and the shadow guard are two copies of one fact. A verb
// added to Dispatch without teaching builtinCommand about it becomes
// shadowable, so this walks the guard against the switch's own answers.
func TestBuiltinCommand_CoversEveryDispatchedVerb(t *testing.T) {
	withExternals(t)
	captureOut(t)

	for _, name := range []string{"charts", "version", "--version", "-v", "help", "--help", "-h"} {
		require.True(t, builtinCommand(name), "%q is dispatched but not guarded", name)
	}
	require.False(t, builtinCommand("license"))
	require.False(t, builtinCommand(""))
}
