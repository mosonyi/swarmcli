// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// envContextVar is the environment variable Docker resolves ahead of the
// config file (docker/cli cli/command.EnvOverrideContext).
const envContextVar = "DOCKER_CONTEXT"

// envContext reports the context named by the environment, if any, trimmed.
// The pin, ValidateContext's refusal guard and the drift check all read it
// through here, so they cannot disagree about what "pinned by the environment"
// means — including for a value that is only whitespace, which names nothing.
func envContext() string { return strings.TrimSpace(os.Getenv(envContextVar)) }

// The Docker context this process addresses, resolved once.
//
// swarmcli used to answer "which context?" twice, differently: the SDK client
// resolved it on first use and cached the connection for the life of the
// process, while every other caller shelled out to `docker context show` again
// and re-read ~/.docker/config.json. A `docker context use` in another terminal
// moved the second answer and not the first, so the lists kept describing one
// swarm while `stack deploy`, `stack rm` and the raw log fallbacks addressed
// another — see #611.
//
// The pin is the single answer. It does not follow the config file: a context
// switch made outside swarmcli must not move a running session between swarms
// under the operator's feet. The app watches for that drift and offers the
// switch instead (app/contextdrift.go).
var (
	sessionCtxMu sync.RWMutex
	sessionCtx   string
)

// showContextFn is the seam for `docker context show`, so the pin's behaviour
// is testable without a Docker CLI on PATH.
var showContextFn = runContextShow

// InitSessionContext resolves the session context and pins it, returning the
// pinned name. Calling it again returns the existing pin without re-resolving,
// so the entry points may both call it (the TUI does, and so does the
// non-interactive CLI) without one of them winning a race.
func InitSessionContext() (string, error) {
	return SessionContext()
}

// SessionContext returns the pinned Docker context, resolving it on first call.
//
// Resolution order is Docker's own, minus the parts a library cannot see:
// $DOCKER_CONTEXT, then the active context from the config file. Note that
// Docker resolves $DOCKER_HOST ahead of $DOCKER_CONTEXT and lets it force the
// `default` context; swarmcli has never honoured $DOCKER_HOST — it always
// builds its client from a context's stored endpoint — which is why every
// shell-out names the pin with an explicit `--context`, the one thing that
// outranks $DOCKER_HOST.
func SessionContext() (string, error) {
	sessionCtxMu.RLock()
	pinned := sessionCtx
	sessionCtxMu.RUnlock()
	if pinned != "" {
		return pinned, nil
	}

	sessionCtxMu.Lock()
	defer sessionCtxMu.Unlock()
	// Another goroutine may have resolved it while this one waited.
	if sessionCtx != "" {
		return sessionCtx, nil
	}

	name, err := resolveContextName()
	if err != nil {
		return "", err
	}
	sessionCtx = name
	return sessionCtx, nil
}

// SetSessionContext moves the pin. It is how a switch made inside swarmcli —
// from the contexts view, or by accepting the drift prompt — takes effect: the
// client cache must be dropped separately with ResetClient, because a pinned
// name and a live connection are two different pieces of state.
func SetSessionContext(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	sessionCtxMu.Lock()
	sessionCtx = name
	sessionCtxMu.Unlock()
}

// EnvPinsContext reports whether DOCKER_CONTEXT names the context.
//
// While it does, ~/.docker/config.json cannot move this session — Docker
// resolves the variable ahead of the file — so the two naming different
// contexts is the documented arrangement rather than a switch to react to.
// ValidateContext refuses to move off it for the same reason.
func EnvPinsContext() bool {
	return envContext() != ""
}

// ResetSessionContext drops the pin, so the next SessionContext call resolves
// afresh.
//
// For tests. The pin is deliberately sticky — that is the whole point of it —
// which is exactly what makes a test that changes DOCKER_CONTEXT or the active
// context read a stale answer without this. Production moves the pin with
// SetSessionContext, which names the new context rather than re-reading the
// config file.
func ResetSessionContext() {
	sessionCtxMu.Lock()
	sessionCtx = ""
	sessionCtxMu.Unlock()
}

// ConfigFileContext reports the context Docker itself would use right now,
// reading past the pin. It is what the drift check compares against, and the
// only caller that wants the live answer rather than the session's.
func ConfigFileContext() (string, error) {
	return resolveContextName()
}

// resolveContextName performs the lookup the pin caches.
func resolveContextName() (string, error) {
	if ctxName := envContext(); ctxName != "" {
		return ctxName, nil
	}
	return showContextFn()
}

func runContextShow() (string, error) {
	out, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get docker context: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
